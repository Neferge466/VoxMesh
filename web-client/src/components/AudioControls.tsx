import { useAudioStore } from '../stores/audioStore';
import { useChannelStore } from '../stores/channelStore';
import { useAuthStore } from '../stores/authStore';
import { useCallback, useEffect, useRef, useState } from 'react';
import { startCapture, stopCapture, onSpeakingChange, setVADParams } from '../api/audioCapture';
import {
  startSending, stopSending, closeAllConnections,
  setupSignaling, teardownSignaling,
} from '../api/webrtc';
import { connectSFU, disconnectSFU, startSFUPublish, stopSFUPublish, isSFUActive } from '../api/livekit';
import { sendWS } from '../api/ws';
import { uuid } from '../lib/uuid';
import * as channelApi from '../api/channels';

const SFU_THRESHOLD = 4;
const SPEAKING_DEBOUNCE_MS = 300;
const PTT_KEY = ' '; // Space bar

export function AudioControls() {
  const { muted, deafened, toggleMute, toggleDeafen, setSpeaking } = useAudioStore();
  const currentId = useChannelStore((s) => s.currentChannelId);
  const leaveChannel = useChannelStore((s) => s.leaveChannel);
  const members = useChannelStore((s) => s.members);
  const memberCount = members.length;
  const token = useAuthStore((s) => s.accessToken);

  const useSFU = memberCount >= SFU_THRESHOLD;
  const modeRef = useRef<'p2p' | 'sfu' | 'none'>('none');
  const speakingTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const lastSpeakingRef = useRef(false);
  const [leaving, setLeaving] = useState(false);
  const [vadThreshold, setVadThreshold] = useState(15);
  const [pttMode, setPttMode] = useState(false);
  const pttActiveRef = useRef(false);

  // PTT: Space bar to push-to-talk
  useEffect(() => {
    if (!pttMode || !currentId) return;

    // In PTT mode, force mute on
    if (!muted) toggleMute();

    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === PTT_KEY && !e.repeat && !pttActiveRef.current) {
        e.preventDefault();
        pttActiveRef.current = true;
        useAudioStore.getState().toggleMute(); // unmute
      }
    };
    const handleKeyUp = (e: KeyboardEvent) => {
      if (e.key === PTT_KEY && pttActiveRef.current) {
        pttActiveRef.current = false;
        useAudioStore.getState().toggleMute(); // mute
      }
    };

    window.addEventListener('keydown', handleKeyDown);
    window.addEventListener('keyup', handleKeyUp);
    return () => {
      window.removeEventListener('keydown', handleKeyDown);
      window.removeEventListener('keyup', handleKeyUp);
      if (pttActiveRef.current) {
        pttActiveRef.current = false;
        if (!useAudioStore.getState().muted) useAudioStore.getState().toggleMute();
      }
    };
  }, [pttMode, currentId]);

  // Speaking detection
  const handleSpeakingChange = useCallback((isSpeaking: boolean) => {
    if (speakingTimerRef.current) {
      clearTimeout(speakingTimerRef.current);
      speakingTimerRef.current = null;
    }
    if (isSpeaking === lastSpeakingRef.current) return;

    speakingTimerRef.current = setTimeout(() => {
      lastSpeakingRef.current = isSpeaking;
      setSpeaking(isSpeaking);
      if (currentId) {
        sendWS({
          type: isSpeaking ? 'start_speaking' : 'stop_speaking',
          id: uuid(),
          timestamp_ms: Date.now(),
          payload: {},
        });
      }
      speakingTimerRef.current = null;
    }, SPEAKING_DEBOUNCE_MS);
  }, [currentId, setSpeaking]);

  useEffect(() => {
    onSpeakingChange(handleSpeakingChange);
    return () => { onSpeakingChange(null); };
  }, [handleSpeakingChange]);

  // Leave channel handler
  const handleLeave = useCallback(async () => {
    if (leaving || !currentId || !token) return;
    setLeaving(true);
    try {
      sendWS({ type: 'leave_channel', id: uuid(), timestamp_ms: Date.now(), payload: { channel_id: currentId } });
      try { await channelApi.leaveChannel(token, currentId); } catch { /* ok */ }
    } finally {
      leaveChannel();
      setLeaving(false);
    }
  }, [currentId, token, leaving, leaveChannel]);

  // Effect 1: signaling + connection mode
  useEffect(() => {
    if (!currentId) {
      disconnectSFU().catch(() => {});
      teardownSignaling();
      closeAllConnections();
      modeRef.current = 'none';
      return;
    }

    if (useSFU) {
      if (modeRef.current === 'sfu' && isSFUActive()) return;
      teardownSignaling();
      closeAllConnections();
      modeRef.current = 'sfu';
      connectSFU(currentId).then(() => {
        if (modeRef.current === 'sfu') {
          if (!muted && !deafened) {
            startCapture().then(() => startSFUPublish()).catch(() => {});
          }
        }
      }).catch((e) => {
        console.error('[sfu] connect failed:', e);
        modeRef.current = 'none';
      });
    } else {
      if (modeRef.current === 'p2p') return;
      disconnectSFU().catch(() => {});
      modeRef.current = 'p2p';
      setupSignaling();
    }

    return () => {
      disconnectSFU().catch(() => {});
      teardownSignaling();
      closeAllConnections();
      modeRef.current = 'none';
    };
  }, [currentId, useSFU]);

  // Effect 2: mute state controls audio send
  const sendingRef = useRef(false);

  useEffect(() => {
    if (!currentId || muted || deafened) {
      stopSending();
      stopSFUPublish().catch(() => {});
      stopCapture();
      sendingRef.current = false;
      return;
    }

    if (sendingRef.current) return;
    sendingRef.current = true;

    startCapture()
      .then(() => {
        if (!sendingRef.current) return;
        if (useSFU) {
          startSFUPublish().catch(() => {});
        } else {
          startSending();
        }
      })
      .catch((err) => {
        console.error('[audio] start failed:', err);
        sendingRef.current = false;
      });

    return () => {
      stopSending();
      stopSFUPublish().catch(() => {});
      stopCapture();
      sendingRef.current = false;
    };
  }, [muted, deafened, currentId, useSFU]);

  // Sync VAD threshold to gate worklet
  useEffect(() => {
    setVADParams(vadThreshold);
  }, [vadThreshold]);

  // Effect 3: cleanup on channel leave
  useEffect(() => {
    return () => {
      closeAllConnections();
      disconnectSFU().catch(() => {});
      stopCapture();
    };
  }, [currentId]);

  if (!currentId) return null;

  return (
    <div className="audio-controls">
      {useSFU && (
        <span className="sfu-badge" title="SFU mode — 4+ members">
          SFU·{memberCount}
        </span>
      )}
      <div className="audio-controls-row">
        <button
          className={`ctrl-btn ${!muted ? 'active' : ''}`}
          onClick={() => { if (!pttMode) toggleMute(); }}
          title={muted ? 'Unmute microphone' : 'Mute microphone'}
        >
          {muted ? (pttMode ? 'Hold Space' : 'Muted') : 'Mic Live'}
          <span className="ctrl-label">Mute</span>
        </button>
        <button
          className={`ctrl-btn ${deafened ? 'active' : ''}`}
          onClick={toggleDeafen}
          title={deafened ? 'Undeafen' : 'Deafen'}
        >
          {deafened ? 'Deafened' : 'Hearing'}
          <span className="ctrl-label">Deafen</span>
        </button>
        <button
          className={`ctrl-btn ${pttMode ? 'active' : ''}`}
          onClick={() => setPttMode((v) => !v)}
          title={pttMode ? 'Disable push-to-talk' : 'Enable push-to-talk (hold Space to speak)'}
        >
          PTT
          <span className="ctrl-label">{pttMode ? 'On' : 'Off'}</span>
        </button>
      </div>
      <div className="audio-controls-row">
        <label className="vad-control" title="Voice activity sensitivity — lower = more sensitive">
          <span>Sens</span>
          <input
            type="range"
            min="2"
            max="80"
            step="0.5"
            value={vadThreshold}
            onChange={(e) => setVadThreshold(parseFloat(e.target.value))}
          />
          <span className="vad-value">{vadThreshold.toFixed(1)}×</span>
        </label>
      </div>
      <button
        className="ctrl-btn leave-btn"
        onClick={handleLeave}
        disabled={leaving}
        title="Leave current channel"
      >
        {leaving ? '...' : 'Leave'}
        <span className="ctrl-label">Channel</span>
      </button>
    </div>
  );
}
