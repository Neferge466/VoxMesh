import { create } from 'zustand';

interface WebRTCState {
  remoteAudioTracks: Map<string, MediaStreamTrack>;
  volumes: Map<string, number>;
  addTrack: (userId: string, track: MediaStreamTrack) => void;
  removeTrack: (userId: string) => void;
  setVolume: (userId: string, volume: number) => void;
  clearAll: () => void;
}

export const useWebRTCStore = create<WebRTCState>((set) => ({
  remoteAudioTracks: new Map(),
  volumes: new Map(),

  addTrack: (userId, track) => {
    set((state) => {
      const next = new Map(state.remoteAudioTracks);
      next.set(userId, track);
      const nextVol = new Map(state.volumes);
      if (!nextVol.has(userId)) nextVol.set(userId, 1.0);
      return { remoteAudioTracks: next, volumes: nextVol };
    });
  },

  removeTrack: (userId) => {
    set((state) => {
      const next = new Map(state.remoteAudioTracks);
      next.delete(userId);
      const nextVol = new Map(state.volumes);
      nextVol.delete(userId);
      return { remoteAudioTracks: next, volumes: nextVol };
    });
  },

  setVolume: (userId, volume) => {
    set((state) => {
      const next = new Map(state.volumes);
      next.set(userId, Math.max(0, Math.min(1, volume)));
      return { volumes: next };
    });
  },

  clearAll: () => {
    set({ remoteAudioTracks: new Map(), volumes: new Map() });
  },
}));
