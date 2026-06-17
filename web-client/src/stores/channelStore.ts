import { create } from 'zustand';

export interface Channel {
  id: string;
  parent_id: string | null;
  name: string;
  description: string;
  sort_order: number;
  member_count: number;
  has_password: boolean;
  codec_quality: string;
  children?: Channel[];
}

export interface Member {
  user_id: string;
  display_name?: string;
  speaking: boolean;
  muted: boolean;
  client_type: string;
}

interface ChannelState {
  channels: Channel[];
  currentChannelId: string | null;
  members: Member[];
  setChannels: (ch: Channel[]) => void;
  joinChannel: (id: string) => void;
  leaveChannel: () => void;
  setMembers: (m: Member[]) => void;
}

export const useChannelStore = create<ChannelState>((set) => ({
  channels: [],
  currentChannelId: null,
  members: [],
  setChannels: (channels) => set({ channels: Array.isArray(channels) ? channels : [] }),
  joinChannel: (currentChannelId) => set({ currentChannelId }),
  leaveChannel: () => set({ currentChannelId: null, members: [] }),
  setMembers: (members) => set({ members }),
}));
