import { create } from 'zustand';

interface AudioState {
  muted: boolean;
  deafened: boolean;
  speaking: boolean;
  toggleMute: () => void;
  toggleDeafen: () => void;
  setSpeaking: (v: boolean) => void;
}

export const useAudioStore = create<AudioState>((set, get) => ({
  muted: true,
  deafened: false,
  speaking: false,

  toggleMute: () => {
    set({ muted: !get().muted });
  },

  toggleDeafen: () => {
    const next = !get().deafened;
    if (next) {
      set({ deafened: true, muted: true });
    } else {
      set({ deafened: false });
    }
  },

  setSpeaking: (speaking) => set({ speaking }),
}));
