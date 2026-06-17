import { create } from 'zustand';

interface WebRTCState {
  remoteAudioTracks: Map<string, MediaStreamTrack>;
  addTrack: (userId: string, track: MediaStreamTrack) => void;
  removeTrack: (userId: string) => void;
  clearAll: () => void;
}

export const useWebRTCStore = create<WebRTCState>((set) => ({
  remoteAudioTracks: new Map(),

  addTrack: (userId, track) => {
    set((state) => {
      const next = new Map(state.remoteAudioTracks);
      next.set(userId, track);
      return { remoteAudioTracks: next };
    });
  },

  removeTrack: (userId) => {
    set((state) => {
      const next = new Map(state.remoteAudioTracks);
      next.delete(userId);
      return { remoteAudioTracks: next };
    });
  },

  clearAll: () => {
    set({ remoteAudioTracks: new Map() });
  },
}));
