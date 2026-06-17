// SFU audio via LiveKit — used for 4+ user channels.
// For ≤3 users, P2P WebRTC mesh (webrtc.ts) is more efficient.
//
// LiveKit client SDK (~120KB gzipped) is only loaded when needed.
// The P2P/SFU switch is automatic based on channel member count.

import { Room, RoomEvent, ConnectionState, type RemoteParticipant } from 'livekit-client';
import { API_HOST } from './host';
import { useAuthStore } from '../stores/authStore';
import { useWebRTCStore } from '../stores/webrtcStore';

let room: Room | null = null;
let localTrack: any = null;

interface LiveKitToken {
  token: string;
  url: string;
  room: string;
  username: string;
}

async function fetchToken(channelId: string): Promise<LiveKitToken> {
  const token = useAuthStore.getState().token;
  if (!token) throw new Error('not authenticated');

  const resp = await fetch(`${API_HOST}/api/v1/system/livekit-token?channel_id=${encodeURIComponent(channelId)}`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  if (!resp.ok) {
    const err = await resp.json().catch(() => null);
    throw new Error(err?.error?.message ?? 'LiveKit token fetch failed');
  }
  return resp.json();
}

export async function connectSFU(channelId: string): Promise<void> {
  if (room?.state === ConnectionState.Connected && room.name === channelId) return;

  // Disconnect any existing room
  await disconnectSFU();

  const tk = await fetchToken(channelId);
  room = new Room({
    adaptiveStream: true,
    dynacast: true,
  });

  room.on(RoomEvent.TrackSubscribed, (_track: any, _pub: any, participant: RemoteParticipant) => {
    console.log('[sfu] track from ' + participant.identity);
    const atPub = participant.audioTrackPublications.values().next().value;
    const track = atPub?.audioTrack?.mediaStreamTrack ?? atPub?.track?.mediaStreamTrack;
    if (track) useWebRTCStore.getState().addTrack(participant.identity, track);
  });

  room.on(RoomEvent.TrackUnsubscribed, (_track: any, _pub: any, participant: RemoteParticipant) => {
    useWebRTCStore.getState().removeTrack(participant.identity);
  });

  room.on(RoomEvent.Disconnected, () => {
    console.log('[sfu] disconnected');
    useWebRTCStore.getState().clearAll();
  });

  await room.connect(tk.url, tk.token);
  console.log('[sfu] connected to room ' + channelId);
}

export async function startSFUPublish(): Promise<void> {
  if (!room) return;
  if (localTrack) return;

  try {
    const { createLocalAudioTrack } = await import('livekit-client');
    localTrack = await createLocalAudioTrack({
      echoCancellation: true,
      noiseSuppression: true,
      autoGainControl: true,
    });
    await room.localParticipant.publishTrack(localTrack);
    console.log('[sfu] publishing audio');
  } catch (e) {
    console.error('[sfu] publish failed:', e);
  }
}

export async function stopSFUPublish(): Promise<void> {
  if (!room || !localTrack) return;
  try {
    await room.localParticipant.unpublishTrack(localTrack);
    localTrack.stop();
    localTrack = null;
    console.log('[sfu] unpublish audio');
  } catch {
    // ignore
  }
}

export async function disconnectSFU(): Promise<void> {
  if (localTrack) {
    try { localTrack.stop(); } catch { /* ok */ }
    localTrack = null;
  }
  if (room) {
    useWebRTCStore.getState().clearAll();
    try { room.disconnect(); } catch { /* ok */ }
    room = null;
  }
}

export function isSFUActive(): boolean {
  return room?.state === ConnectionState.Connected;
}
