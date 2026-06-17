import { useAudioStore } from '../stores/audioStore';
import { useChannelStore } from '../stores/channelStore';
import { useEffect, useRef } from 'react';
import { startCapture, stopCapture } from '../api/audioCapture';
import { startSending, stopSending, closeAllConnections, setupSignaling, teardownSignaling } from '../api/webrtc';

export function AudioControls() {
  const { muted, deafened, toggleMute, toggleDeafen } = useAudioStore();
  const currentId = useChannelStore((s) => s.currentChannelId);

  // Effect 1: signaling listeners — stay alive as long as we're in a channel
  useEffect(() => {
    if (!currentId) {
      teardownSignaling();
      return;
    }
    setupSignaling();
    return () => { teardownSignaling(); };
  }, [currentId]);

  // Effect 2: mute state controls capture + sending.
  // Peer connections stay open (for receiving) even when muted.
  const sendingRef = useRef(false);

  useEffect(() => {
    if (!currentId || muted) {
      stopSending();
      stopCapture();
      sendingRef.current = false;
      return;
    }

    if (sendingRef.current) return;
    sendingRef.current = true;

    startCapture()
      .then(() => {
        if (!sendingRef.current) return;
        startSending();
      })
      .catch((err) => {
        console.error('[audio] start failed:', err);
        sendingRef.current = false;
      });

    return () => {
      stopSending();
      stopCapture();
      sendingRef.current = false;
    };
  }, [muted, currentId]);

  // Effect 3: leave channel → close all WebRTC connections
  useEffect(() => {
    return () => {
      closeAllConnections();
      stopCapture();
    };
  }, [currentId]);

  if (!currentId) return null;

  return (
    <div className="audio-controls">
      <button
        className={`ctrl-btn ${!muted ? 'active' : ''}`}
        onClick={toggleMute}
        title={muted ? 'Unmute microphone' : 'Mute microphone'}
      >
        {muted ? 'Muted' : 'Mic Live'}
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
    </div>
  );
}
