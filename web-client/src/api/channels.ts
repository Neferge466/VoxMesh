import { API_HOST } from './host';
const BASE = `${API_HOST}/api/v1/channels`;

export interface Channel {
  id: string;
  parent_id: string | null;
  name: string;
  description: string;
  sort_order: number;
  max_users: number;
  member_count: number;
  has_password: boolean;
  codec_quality: string;
  children?: Channel[];
}

function authHeaders(token: string): Record<string, string> {
  return {
    'Content-Type': 'application/json',
    Authorization: `Bearer ${token}`,
  };
}

export async function listChannels(token: string): Promise<Channel[]> {
  const res = await fetch(BASE, { headers: authHeaders(token) });
  if (!res.ok) {
    const body = await res.json().catch(() => null);
    throw new Error(body?.error?.message ?? res.statusText);
  }
  const data = await res.json();
  return Array.isArray(data) ? data : [];
}

export async function createChannel(
  token: string,
  name: string,
  password?: string,
): Promise<Channel> {
  const body: Record<string, unknown> = { name };
  if (password) body.password = password;
  const res = await fetch(BASE, {
    method: 'POST',
    headers: authHeaders(token),
    body: JSON.stringify(body),
  });
  if (!res.ok) {
    const err = await res.json().catch(() => null);
    throw new Error(err?.error?.message ?? res.statusText);
  }
  return res.json();
}

export async function joinChannel(
  token: string,
  channelId: string,
  password?: string,
  signal?: AbortSignal,
) {
  const body: Record<string, unknown> = {};
  if (password) body.password = password;
  const res = await fetch(`${BASE}/${channelId}/join`, {
    method: 'POST',
    headers: authHeaders(token),
    body: JSON.stringify(body),
    signal,
  });
  if (!res.ok) {
    const err = await res.json().catch(() => null);
    throw new Error(err?.error?.message ?? res.statusText);
  }
  return res.json();
}

export async function leaveChannel(token: string, channelId: string) {
  const res = await fetch(`${BASE}/${channelId}/leave`, {
    method: 'POST',
    headers: authHeaders(token),
  });
  if (!res.ok) {
    const body = await res.json().catch(() => null);
    throw new Error(body?.error?.message ?? res.statusText);
  }
  return res.json();
}

export async function kickUser(token: string, channelId: string, userId: string) {
  const res = await fetch(`${BASE}/${channelId}/kick/${userId}`, {
    method: 'POST',
    headers: authHeaders(token),
  });
  if (!res.ok) {
    const err = await res.json().catch(() => null);
    throw new Error(err?.error?.message ?? res.statusText);
  }
  return res.json();
}

export interface Member {
  id: string;
  user_id: string;
  channel_id: string;
  display_name: string;
  client_type: string;
  device_id: string | null;
  joined_at: string;
  left_at: string | null;
}

export async function getMembers(token: string, channelId: string): Promise<Member[]> {
  const res = await fetch(`${BASE}/${channelId}/members`, { headers: authHeaders(token) });
  if (!res.ok) {
    const body = await res.json().catch(() => null);
    throw new Error(body?.error?.message ?? res.statusText);
  }
  return res.json();
}
