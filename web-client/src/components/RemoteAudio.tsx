import { useEffect, useRef } from 'react';
import { useWebRTCStore } from '../stores/webrtcStore';

export function RemoteAudio() {
  const tracks = useWebRTCStore((s) => s.remoteAudioTracks);
  const volumes = useWebRTCStore((s) => s.volumes);
  const streamsRef = useRef<Map<string, MediaStream>>(new Map());
  const audioRef = useRef<Map<string, HTMLAudioElement>>(new Map());

  // Create/remove audio elements when tracks change
  useEffect(() => {
    const currentStreams = streamsRef.current;
    const audioEls = audioRef.current;

    tracks.forEach((track, userId) => {
      if (!currentStreams.has(userId)) {
        const stream = new MediaStream([track]);
        currentStreams.set(userId, stream);

        const audio = new Audio();
        audio.autoplay = true;
        audio.srcObject = stream;
        audio.volume = volumes.get(userId) ?? 1.0;
        audio.play().catch((e) => console.warn('[webrtc] autoplay blocked for ' + userId, e));
        audioEls.set(userId, audio);
      }
    });

    currentStreams.forEach((stream, userId) => {
      if (!tracks.has(userId)) {
        stream.getTracks().forEach((t) => t.stop());
        currentStreams.delete(userId);
        const audio = audioEls.get(userId);
        if (audio) { audio.srcObject = null; audio.remove(); }
        audioEls.delete(userId);
      }
    });

    return () => {
      currentStreams.forEach((stream) => {
        stream.getTracks().forEach((t) => t.stop());
      });
      currentStreams.clear();
      audioEls.forEach((audio) => { audio.srcObject = null; audio.remove(); });
      audioEls.clear();
    };
  }, [tracks]);

  // Sync volume changes to audio elements
  useEffect(() => {
    const audioEls = audioRef.current;
    volumes.forEach((vol, userId) => {
      const audio = audioEls.get(userId);
      if (audio) audio.volume = vol;
    });
  }, [volumes]);

  return null;
}
