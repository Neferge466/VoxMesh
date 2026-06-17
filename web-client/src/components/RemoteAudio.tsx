import { useEffect, useRef } from 'react';
import { useWebRTCStore } from '../stores/webrtcStore';

export function RemoteAudio() {
  const tracks = useWebRTCStore((s) => s.remoteAudioTracks);
  const streamsRef = useRef<Map<string, MediaStream>>(new Map());

  useEffect(() => {
    const currentStreams = streamsRef.current;

    // Add new tracks
    tracks.forEach((track, userId) => {
      if (!currentStreams.has(userId)) {
        const stream = new MediaStream([track]);
        currentStreams.set(userId, stream);

        const audio = new Audio();
        audio.autoplay = true;
        audio.srcObject = stream;
        audio.play().catch((e) => console.warn('[webrtc] autoplay blocked for ' + userId, e));

        // Store the audio element so it doesn't get garbage collected
        (audio as any).__webrtc_user = userId;
      }
    });

    // Remove stale tracks
    currentStreams.forEach((stream, userId) => {
      if (!tracks.has(userId)) {
        stream.getTracks().forEach((t) => t.stop());
        currentStreams.delete(userId);
      }
    });

    // Cleanup on unmount
    return () => {
      currentStreams.forEach((stream) => {
        stream.getTracks().forEach((t) => t.stop());
      });
      currentStreams.clear();
    };
  }, [tracks]);

  // This component renders nothing — audio is played via Audio elements
  return null;
}
