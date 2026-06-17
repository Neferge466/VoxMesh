import { uuid } from '../lib/uuid';
import { sendWS, onWS } from './ws';
import { useAuthStore } from '../stores/authStore';
import { useWebRTCStore } from '../stores/webrtcStore';
import { getLocalStream } from './audioCapture';

const ICE_CONFIG: RTCConfiguration = {
  iceServers: [{ urls: 'stun:stun.l.google.com:19302' }],
};

const peerConnections = new Map<string, RTCPeerConnection>();
let pendingOfferPC: RTCPeerConnection | null = null;
let unsubs: Array<() => void> = [];
let signalingActive = false;
let sendingActive = false;

function getUserId(): string | undefined {
  return useAuthStore.getState().user?.id;
}

function createPeerConnection(remoteUserId: string): RTCPeerConnection {
  const pc = new RTCPeerConnection(ICE_CONFIG);

  pc.onicecandidate = (event) => {
    if (event.candidate) {
      const uid = getUserId();
      if (!uid) return;
      sendWS({
        type: 'ice_candidate',
        id: uuid(),
        timestamp_ms: Date.now(),
        payload: { candidate: JSON.stringify(event.candidate), sender_id: uid },
      });
    }
  };

  pc.ontrack = (event) => {
    const track = event.track;
    if (track.kind === 'audio') {
      console.log('[webrtc] remote track from user=' + remoteUserId);
      useWebRTCStore.getState().addTrack(remoteUserId, track);
    }
  };

  pc.oniceconnectionstatechange = () => {
    console.log('[webrtc] ICE ' + remoteUserId + ': ' + pc.iceConnectionState);
  };

  // Automatic renegotiation when tracks are added/removed
  let negotiating = false;
  pc.onnegotiationneeded = async () => {
    if (negotiating) return;
    negotiating = true;
    console.log('[webrtc] negotiation needed for ' + remoteUserId);
    try {
      const offer = await pc.createOffer();
      await pc.setLocalDescription(offer);
      const uid = getUserId();
      if (!uid) return;
      sendWS({
        type: 'sdp_offer',
        id: uuid(),
        timestamp_ms: Date.now(),
        payload: { sdp: offer.sdp, sender_id: uid },
      });
    } catch (e) {
      console.error('[webrtc] renegotiation failed:', e);
    } finally {
      negotiating = false;
    }
  };

  return pc;
}

export function setupSignaling(): void {
  if (signalingActive) return;
  signalingActive = true;
  console.log('[webrtc] signaling registered');
  unsubs = [
    onWS('sdp_offer', (msg: any) => handleMeshSignal(msg)),
    onWS('sdp_answer', (msg: any) => handleMeshSignal(msg)),
    onWS('ice_candidate', (msg: any) => handleMeshSignal(msg)),
  ];
}

export function teardownSignaling(): void {
  if (!signalingActive) return;
  signalingActive = false;
  console.log('[webrtc] signaling removed');
  unsubs.forEach((u) => u());
  unsubs = [];
}

// Start sending audio — add local tracks to all peer connections.
// Let onnegotiationneeded handle the SDP exchange.
export async function startSending(): Promise<void> {
  if (sendingActive) return;

  const stream = getLocalStream();
  if (!stream) return;

  const userId = getUserId();
  if (!userId) return;

  sendingActive = true;
  const tracks = stream.getAudioTracks();
  console.log('[webrtc] start sending, peers=' + peerConnections.size + ' pending=' + !!pendingOfferPC);

  if (peerConnections.size > 0) {
    for (const [remoteId, pc] of peerConnections) {
      for (const t of tracks) {
        if (!pc.getSenders().find((s) => s.track === t)) {
          pc.addTrack(t, stream);
        }
      }
    }
  } else {
    // No peers yet — create pending PC, addTrack triggers onnegotiationneeded
    const pc = createPeerConnection('pending');
    pendingOfferPC = pc;
    for (const t of tracks) pc.addTrack(t, stream);
  }
}

// Stop sending audio — replace local tracks with null.
// Peer connections stay alive so we can keep receiving.
export function stopSending(): void {
  if (!sendingActive) return;
  sendingActive = false;
  console.log('[webrtc] stop sending');

  for (const [, pc] of peerConnections) {
    for (const sender of pc.getSenders()) {
      if (sender.track?.kind === 'audio') {
        sender.replaceTrack(null).catch(() => {});
      }
    }
  }
}

// Close all connections — called on channel leave.
export function closeAllConnections(): void {
  sendingActive = false;
  console.log('[webrtc] close all, connections=' + peerConnections.size);

  useWebRTCStore.getState().clearAll();

  if (pendingOfferPC) {
    pendingOfferPC.close();
    pendingOfferPC = null;
  }
  for (const [, pc] of peerConnections) pc.close();
  peerConnections.clear();
}

async function handleMeshSignal(msg: any): Promise<void> {
  const payload = msg.payload;
  const remoteUserId = payload.sender_id;
  const userId = getUserId();
  if (!userId) return;
  if (remoteUserId === userId) return;

  const type = msg.type as string;
  console.log('[webrtc] signal: ' + type + ' from=' + remoteUserId);

  if (type === 'sdp_offer') {
    let pc = peerConnections.get(remoteUserId);
    if (!pc) {
      pc = createPeerConnection(remoteUserId);
      peerConnections.set(remoteUserId, pc);
    }

    // Add local tracks if we're currently sending
    const stream = getLocalStream();
    if (stream && sendingActive) {
      for (const t of stream.getAudioTracks()) {
        if (!pc.getSenders().find((s) => s.track === t)) {
          pc.addTrack(t, stream);
        }
      }
    }

    try {
      await pc.setRemoteDescription(new RTCSessionDescription({ type: 'offer', sdp: payload.sdp }));
      const answer = await pc.createAnswer();
      await pc.setLocalDescription(answer);
      sendWS({
        type: 'sdp_answer',
        id: uuid(),
        timestamp_ms: Date.now(),
        payload: { sdp: answer.sdp, sender_id: remoteUserId },
      });
      console.log('[webrtc] answer sent to ' + remoteUserId);
    } catch (e) {
      console.error('[webrtc] offer handling failed:', e);
    }
  } else if (type === 'sdp_answer') {
    let pc = peerConnections.get(remoteUserId);
    if (!pc) {
      pc = pendingOfferPC;
      if (pc) {
        pendingOfferPC = null;
        peerConnections.set(remoteUserId, pc);
        console.log('[webrtc] pending PC resolved to ' + remoteUserId);
      }
    }
    if (!pc) {
      console.warn('[webrtc] answer from unknown: ' + remoteUserId);
      return;
    }

    try {
      await pc.setRemoteDescription(new RTCSessionDescription({ type: 'answer', sdp: payload.sdp }));
      console.log('[webrtc] connected with ' + remoteUserId);
    } catch (e) {
      console.error('[webrtc] answer failed:', e);
    }
  } else if (type === 'ice_candidate') {
    let pc = peerConnections.get(remoteUserId);
    if (!pc) pc = pendingOfferPC;
    if (!pc) return;

    try {
      const candidate = JSON.parse(payload.candidate) as RTCIceCandidateInit;
      await pc.addIceCandidate(candidate);
    } catch (e) {
      console.error('[webrtc] ICE failed:', e);
    }
  }
}
