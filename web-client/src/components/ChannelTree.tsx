import { useState, type FormEvent } from 'react';
import { useChannelStore, type Channel } from '../stores/channelStore';
import * as channelApi from '../api/channels';
import { sendWS, setActiveChannel } from '../api/ws';
import { uuid } from '../lib/uuid';

function ChannelItem({ ch, depth = 0, token }: { ch: Channel; depth?: number; token: string }) {
  const currentId = useChannelStore((s) => s.currentChannelId);
  const joinChannel = useChannelStore((s) => s.joinChannel);
  const setMembers = useChannelStore((s) => s.setMembers);
  const isActive = currentId === ch.id;

  async function handleClick() {
    if (isActive) return;
    const prevChannelId = currentId;
    try {
      // Leave old channel (REST + WebSocket)
      if (prevChannelId) {
        try { await channelApi.leaveChannel(token, prevChannelId); } catch { /* ok */ }
        sendWS({ type: 'leave_channel', id: uuid(), timestamp_ms: Date.now(), payload: { channel_id: prevChannelId } });
      }
      // Join new channel via REST (already-in-channel from stale session is OK)
      try { await channelApi.joinChannel(token, ch.id); } catch (e: any) { /* already-in-channel is OK */ }
      // Update store and WebSocket regardless of join result
      joinChannel(ch.id);
      setActiveChannel(ch.id);
      // Load members
      const raw = await channelApi.getMembers(token, ch.id);
      setMembers(raw.map((m) => ({
        user_id: m.user_id,
        display_name: (m as Record<string, unknown>).display_name as string,
        client_type: m.client_type,
        speaking: false,
        muted: false,
      })));
    } catch (err) {
      console.error('[channel] join failed:', err);
    }
  }

  return (
    <div>
      <button
        className={`channel-item ${isActive ? 'active' : ''}`}
        style={{ paddingLeft: `calc(var(--s-4) + ${depth * 20}px)` }}
        onClick={handleClick}
      >
        <span className="channel-icon">{ch.children?.length ? '▾' : '#'}</span>
        <span className="channel-name">{ch.name}</span>
        <span className="channel-count">{ch.member_count}</span>
        {ch.has_password && <span className="channel-lock">🔒</span>}
      </button>
      {ch.children?.map((child) => (
        <ChannelItem key={child.id} ch={child} depth={depth + 1} token={token} />
      ))}
    </div>
  );
}

interface Props {
  token: string;
}

export function ChannelTree({ token }: Props) {
  const channels = useChannelStore((s) => s.channels);
  const setChannels = useChannelStore((s) => s.setChannels);
  const joinChannel = useChannelStore((s) => s.joinChannel);
  const setMembers = useChannelStore((s) => s.setMembers);
  const [newName, setNewName] = useState('');
  const [creating, setCreating] = useState(false);
  const [error, setError] = useState('');

  async function handleCreate(e: FormEvent) {
    e.preventDefault();
    const name = newName.trim();
    if (!name) return;
    setCreating(true);
    setError('');
    try {
      const ch = await channelApi.createChannel(token, name);
      setNewName('');
      // auto-join and refresh
      try { await channelApi.joinChannel(token, ch.id); } catch { /* ok */ }
      joinChannel(ch.id);
      setActiveChannel(ch.id);
      const raw = await channelApi.getMembers(token, ch.id);
      setMembers(raw.map((m) => ({
        user_id: m.user_id,
        display_name: (m as Record<string, unknown>).display_name as string,
        client_type: m.client_type,
        speaking: false,
        muted: false,
      })));
      const list = await channelApi.listChannels(token);
      setChannels(list);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Create failed');
    } finally {
      setCreating(false);
    }
  }

  return (
    <nav className="channel-tree" aria-label="Channel list">
      <div className="channel-tree-header">
        <h2 className="channel-tree-title">Channels</h2>
      </div>
      <div className="channel-list">
        {channels.filter((ch, i, arr) => arr.findIndex(x => x.id === ch.id) === i).map((ch) => (
          <ChannelItem key={ch.id} ch={ch} token={token} />
        ))}
        {channels.length === 0 && (
          <p className="channel-empty">No channels yet — create one below</p>
        )}
      </div>
      <form className="channel-create" onSubmit={handleCreate}>
        <input
          type="text"
          value={newName}
          onChange={(e) => setNewName(e.target.value)}
          placeholder="New channel name"
          required
          maxLength={64}
        />
        <button type="submit" disabled={creating}>
          {creating ? '...' : '+'}
        </button>
      </form>
      {error && <p className="channel-error">{error}</p>}
    </nav>
  );
}
