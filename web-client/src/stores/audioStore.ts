import { create } from 'zustand';
import { sendWS } from '../api/ws';
import { uuid } from '../lib/uuid';

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
    const next = !get().muted;
    set({ muted: next });
    sendWS({
      type: 'set_mute',
      id: uuid(),
      timestamp_ms: Date.now(),
      payload: {},
    });
  },

  toggleDeafen: () => {
    const next = !get().deafened;
    if (next) {
      set({ deafened: true, muted: true });
    } else {
      set({ deafened: false });
    }
    sendWS({
      type: 'set_deafen',
      id: uuid(),
      timestamp_ms: Date.now(),
      payload: {},
    });
  },

  setSpeaking: (speaking) => set({ speaking }),
}));
