import { uuid } from '../lib/uuid';
import { sendWS, onWS } from './ws';
import { API_HOST } from './host';
import { useAuthStore } from '../stores/authStore';
import { useWebRTCStore } from '../stores/webrtcStore';
import { getLocalStream } from './audioCapture';

interface TURNCredentials {
  username: string;
  password: string;
  ttl: number;
  stun_uri: string;
  turn_uri: string;
  turns_uri: string;
}

let iceConfig: RTCConfiguration = {
  iceServers: [{ urls: 'stun:stun.l.google.com:19302' }],
};

let turnCreds: TURNCredentials | null = null;
let turnFetchPromise: Promise<void> | null = null;

async function fetchTURNCredentials(): Promise<void> {
  const token = useAuthStore.getState().token;
  if (!token) return;
  try {
    const resp = await fetch(`${API_HOST}/api/v1/system/turn`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    if (resp.ok) {
      turnCreds = await resp.json();
      iceConfig = {
        iceServers: [
          { urls: 'stun:stun.l.google.com:19302' },
          {
            urls: [turnCreds.turn_uri, turnCreds.turns_uri],
            username: turnCreds.username,
            credential: turnCreds.password,
          },
        ],
      };
      console.log('[webrtc] TURN credentials loaded');
    }
  } catch (e) {
    console.warn('[webrtc] TURN fetch failed, using STUN only:', e);
  }
}

function getIceConfig(): RTCConfiguration {
  return iceConfig;
}

interface PeerState {
  pc: RTCPeerConnection;
  polite: boolean;
  makingOffer: boolean;
}

const peers = new Map<string, PeerState>();
let unsubs: Array<() => void> = [];
let signalingActive = false;
let sendingActive = false;

function getUserId(): string | undefined {
  return useAuthStore.getState().user?.id;
}

// Polite peer: lower user ID yields in collision (standard perfect-negotiation rule).
function isPolite(ourId: string, peerId: string): boolean {
  return ourId < peerId;
}

function createPeerState(remoteUserId: string): PeerState {
  const ourId = getUserId()!;
  const state: PeerState = {
    pc: new RTCPeerConnection(getIceConfig()),
    polite: isPolite(ourId, remoteUserId),
    makingOffer: false,
  };

  state.pc.onicecandidate = (event) => {
    if (event.candidate) {
      sendWS({
        type: 'ice_candidate',
        id: uuid(),
        timestamp_ms: Date.now(),
        payload: { candidate: JSON.stringify(event.candidate), sender_id: ourId },
      });
    }
  };

  state.pc.ontrack = (event) => {
    const track = event.track;
    if (track.kind === 'audio') {
      console.log('[webrtc] remote track from ' + remoteUserId);
      useWebRTCStore.getState().addTrack(remoteUserId, track);
    }
  };

  state.pc.oniceconnectionstatechange = () => {
    console.log('[webrtc] ICE ' + remoteUserId + ': ' + state.pc.iceConnectionState);
    if (state.pc.iceConnectionState === 'failed') {
      state.pc.close();
      peers.delete(remoteUserId);
      useWebRTCStore.getState().removeTrack(remoteUserId);
    }
  };

  // Perfect-negotiation: onnegotiationneeded sends an offer.
  // makingOffer prevents re-entry; polite/impolite resolve collisions.
  state.pc.onnegotiationneeded = async () => {
    if (state.makingOffer) return;
    state.makingOffer = true;
    try {
      await state.pc.setLocalDescription();
      sendWS({
        type: 'sdp_offer',
        id: uuid(),
        timestamp_ms: Date.now(),
        payload: { sdp: state.pc.localDescription!.sdp, sender_id: ourId },
      });
    } catch (e) {
      console.error('[webrtc] offer failed for ' + remoteUserId + ':', e);
    } finally {
      state.makingOffer = false;
    }
  };

  return state;
}

// Sync local peer connections to match a member list (called on channel_joined / presence_update).
export function syncPeers(memberUserIds: string[]): void {
  const myId = getUserId();
  if (!myId) return;

  const memberSet = new Set(memberUserIds.filter((id) => id !== myId));

  // Remove stale peers
  for (const [id, state] of peers) {
    if (!memberSet.has(id)) {
      console.log('[webrtc] removing peer: ' + id);
      state.pc.close();
      peers.delete(id);
      useWebRTCStore.getState().removeTrack(id);
    }
  }

  // Create peers for new members
  const stream = getLocalStream();
  for (const id of memberSet) {
    if (!peers.has(id)) {
      console.log('[webrtc] adding peer: ' + id);
      const state = createPeerState(id);
      peers.set(id, state);

      // If we're already sending, add tracks immediately (triggers onnegotiationneeded)
      if (sendingActive && stream) {
        for (const t of stream.getAudioTracks()) {
          state.pc.addTrack(t, stream);
        }
      }
    }
  }
}

export function setupSignaling(): void {
  if (signalingActive) return;
  signalingActive = true;
  console.log('[webrtc] signaling registered');

  // Fetch TURN credentials in background
  if (!turnCreds) {
    turnFetchPromise = fetchTURNCredentials();
  }

  unsubs = [
    onWS('sdp_offer', (msg: any) => handleMeshSignal(msg)),
    onWS('sdp_answer', (msg: any) => handleMeshSignal(msg)),
    onWS('ice_candidate', (msg: any) => handleMeshSignal(msg)),
    // Auto-sync peers on membership updates
    onWS('channel_joined', (msg: any) => {
      const members: Array<{ user_id: string }> | undefined = msg.payload?.members;
      if (members?.length) syncPeers(members.map((m) => m.user_id));
    }),
    onWS('presence_update', (msg: any) => {
      const members: Array<{ user_id: string }> | undefined = msg.payload?.members;
      if (members?.length) syncPeers(members.map((m) => m.user_id));
    }),
  ];
}

export function teardownSignaling(): void {
  if (!signalingActive) return;
  signalingActive = false;
  console.log('[webrtc] signaling removed');
  unsubs.forEach((u) => u());
  unsubs = [];
}

// Start sending audio — add local tracks to all existing peers.
// Peer connections are already created by syncPeers (called on channel_joined).
export async function startSending(): Promise<void> {
  if (sendingActive) return;
  const stream = getLocalStream();
  if (!stream) return;
  sendingActive = true;

  const tracks = stream.getAudioTracks();
  console.log('[webrtc] start sending to ' + peers.size + ' peers');

  if (peers.size === 0) {
    // No peers yet — tracks will be added when syncPeers detects new members.
    console.log('[webrtc] no peers yet, waiting for presence update');
    return;
  }

  for (const [, state] of peers) {
    for (const t of tracks) {
      if (!state.pc.getSenders().find((s) => s.track === t)) {
        state.pc.addTrack(t, stream);
      }
    }
  }
}

// Stop sending — replace local tracks with null. PCs stay alive for receiving.
export function stopSending(): void {
  if (!sendingActive) return;
  sendingActive = false;
  console.log('[webrtc] stop sending');
  for (const [, state] of peers) {
    for (const sender of state.pc.getSenders()) {
      if (sender.track?.kind === 'audio') {
        sender.replaceTrack(null).catch(() => {});
      }
    }
  }
}

// Close all connections — called on channel leave.
export function closeAllConnections(): void {
  sendingActive = false;
  console.log('[webrtc] close all, peers=' + peers.size);
  useWebRTCStore.getState().clearAll();
  for (const [, state] of peers) state.pc.close();
  peers.clear();
}

async function handleMeshSignal(msg: any): Promise<void> {
  const payload = msg.payload;
  const remoteUserId = payload.sender_id;
  const userId = getUserId();
  if (!userId) return;
  if (remoteUserId === userId) return;

  const type = msg.type as string;

  if (type === 'sdp_offer') {
    let state = peers.get(remoteUserId);

    // Perfect-negotiation collision check
    const polite = isPolite(userId, remoteUserId);
    const collision = state?.makingOffer ?? false;

    if (!state) {
      state = createPeerState(remoteUserId);
      peers.set(remoteUserId, state);
    }

    if (collision && !polite) {
      // We are impolite and already sending an offer → ignore incoming.
      // Our offer wins; the other (polite) side will yield.
      console.log('[webrtc] ignoring offer from ' + remoteUserId + ' (impolite, we offered first)');
      return;
    }
    if (collision && polite) {
      // We are polite → roll back our local description and accept incoming.
      console.log('[webrtc] yielding to offer from ' + remoteUserId + ' (polite)');
      try {
        await state.pc.setLocalDescription({ type: 'rollback' });
      } catch {
        // Rollback failed — close and recreate
        state.pc.close();
        state = createPeerState(remoteUserId);
        peers.set(remoteUserId, state);
      }
    }

    try {
      await state.pc.setRemoteDescription(new RTCSessionDescription({ type: 'offer', sdp: payload.sdp }));

      // Add local tracks if sending
      const stream = getLocalStream();
      if (stream && sendingActive) {
        for (const t of stream.getAudioTracks()) {
          if (!state.pc.getSenders().find((s) => s.track === t)) {
            state.pc.addTrack(t, stream);
          }
        }
      }

      const answer = await state.pc.createAnswer();
      await state.pc.setLocalDescription(answer);
      sendWS({
        type: 'sdp_answer',
        id: uuid(),
        timestamp_ms: Date.now(),
        payload: { sdp: answer.sdp, sender_id: remoteUserId },
      });
    } catch (e) {
      console.error('[webrtc] offer handling failed for ' + remoteUserId + ':', e);
    }
  } else if (type === 'sdp_answer') {
    const state = peers.get(remoteUserId);
    if (!state) {
      console.warn('[webrtc] answer from unknown peer: ' + remoteUserId);
      return;
    }
    try {
      await state.pc.setRemoteDescription(
        new RTCSessionDescription({ type: 'answer', sdp: payload.sdp }),
      );
      console.log('[webrtc] connected with ' + remoteUserId);
    } catch (e) {
      console.error('[webrtc] answer failed for ' + remoteUserId + ':', e);
    }
  } else if (type === 'ice_candidate') {
    const state = peers.get(remoteUserId);
    if (!state) return;
    try {
      const candidate = JSON.parse(payload.candidate) as RTCIceCandidateInit;
      await state.pc.addIceCandidate(candidate);
    } catch (e) {
      console.error('[webrtc] ICE failed for ' + remoteUserId + ':', e);
    }
  }
}
