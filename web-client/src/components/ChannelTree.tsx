import { useState, type FormEvent } from 'react';
import { useChannelStore, type Channel } from '../stores/channelStore';
import { useAuthStore } from '../stores/authStore';
import * as channelApi from '../api/channels';
import { sendWS, setActiveChannel } from '../api/ws';
import { uuid } from '../lib/uuid';

const JOIN_TIMEOUT_MS = 15000;

function ChannelItem({ ch, depth = 0, token }: { ch: Channel; depth?: number; token: string }) {
  const currentId = useChannelStore((s) => s.currentChannelId);
  const joinChannel = useChannelStore((s) => s.joinChannel);
  const setMembers = useChannelStore((s) => s.setMembers);
  const isActive = currentId === ch.id;
  const [joining, setJoining] = useState(false);
  const [showPwPrompt, setShowPwPrompt] = useState(false);
  const [pwValue, setPwValue] = useState('');
  const [pwError, setPwError] = useState('');

  const doJoin = async (password?: string) => {
    if (isActive || joining) return;
    if (ch.has_password && password === undefined) {
      setShowPwPrompt(true);
      return;
    }
    setJoining(true);
    setPwError('');

    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), JOIN_TIMEOUT_MS);

    const prevChannelId = currentId;
    try {
      try {
        await channelApi.joinChannel(token, ch.id, password, controller.signal);
      } catch (e: any) {
        const msg: string = e?.message ?? '';
        if (msg.toLowerCase().includes('password') || msg.toLowerCase().includes('locked')) {
          setShowPwPrompt(true);
          setPwError(msg);
          setJoining(false);
          return;
        }
        if (e?.name === 'AbortError') {
          setPwError('Request timed out — please try again');
          setJoining(false);
          return;
        }
        if (msg.toLowerCase().includes('already in')) {
          // Already a member — proceed to update UI state without re-joining
        } else {
          throw e;
        }
      }
      // Join succeeded — now leave old channel
      if (prevChannelId) {
        try { await channelApi.leaveChannel(token, prevChannelId); } catch { /* ok */ }
        sendWS({ type: 'leave_channel', id: uuid(), timestamp_ms: Date.now(), payload: { channel_id: prevChannelId } });
      }
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
      setShowPwPrompt(false);
      setPwValue('');
      setPwError('');
    } catch (err) {
      console.error('[channel] join failed:', err);
      setPwError(err instanceof Error ? err.message : 'Join failed');
    } finally {
      clearTimeout(timeoutId);
      setJoining(false);
    }
  };

  const handlePwSubmit = (e: FormEvent) => {
    e.preventDefault();
    const trimmed = pwValue.trim();
    if (!trimmed) {
      setPwError('Please enter a password');
      return;
    }
    setPwError('');
    doJoin(trimmed);
  };

  return (
    <div>
      <button
        className={`channel-item ${isActive ? 'active' : ''}`}
        style={{ paddingLeft: `calc(var(--s-4) + ${depth * 20}px)` }}
        onClick={() => doJoin()}
        disabled={joining}
      >
        <span className="channel-icon">{ch.children?.length ? '▾' : '#'}</span>
        <span className="channel-name">{ch.name}</span>
        <span className="channel-count">{ch.member_count}</span>
        {ch.has_password && <span className="channel-lock" title="Password protected">🔒</span>}
      </button>

      {pwError && !showPwPrompt && (
        <p className="channel-error" style={{ paddingLeft: `calc(var(--s-4) + ${(depth + 1) * 20}px)` }}>{pwError}</p>
      )}
      {showPwPrompt && (
        <form className="channel-pw-prompt" onSubmit={handlePwSubmit} style={{ paddingLeft: `calc(var(--s-4) + ${(depth + 1) * 20}px)` }}>
          <input
            type="password"
            value={pwValue}
            onChange={(e) => { setPwValue(e.target.value); setPwError(''); }}
            placeholder="Channel password"
            autoFocus
            maxLength={64}
            autoComplete="new-password"
          />
          <button type="submit" disabled={joining}>{joining ? '...' : 'Join'}</button>
          <button type="button" onClick={() => { setShowPwPrompt(false); setPwError(''); setPwValue(''); }}>✕</button>
          {pwError && <p className="channel-error">{pwError}</p>}
        </form>
      )}

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
  const user = useAuthStore((s) => s.user);
  const [newName, setNewName] = useState('');
  const [newPw, setNewPw] = useState('');
  const [creating, setCreating] = useState(false);
  const [error, setError] = useState('');

  async function handleCreate(e: FormEvent) {
    e.preventDefault();
    const name = newName.trim();
    if (!name) return;
    setCreating(true);
    setError('');
    const currentId = useChannelStore.getState().currentChannelId;
    try {
      const ch = await channelApi.createChannel(token, name, newPw || undefined);
      setNewName('');
      setNewPw('');
      if (currentId) {
        try { await channelApi.leaveChannel(token, currentId); } catch { /* ok */ }
        sendWS({ type: 'leave_channel', id: uuid(), timestamp_ms: Date.now(), payload: { channel_id: currentId } });
      }
      try { await channelApi.joinChannel(token, ch.id, newPw || undefined); } catch { /* ok */ }
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
      <form className="channel-create" onSubmit={handleCreate} autoComplete="off">
        <input
          id="channel-create-input"
          type="text"
          value={newName}
          onChange={(e) => setNewName(e.target.value)}
          placeholder="New channel name"
          required
          maxLength={64}
          autoComplete="off"
        />
        <input
          type="password"
          value={newPw}
          onChange={(e) => setNewPw(e.target.value)}
          placeholder="Password (optional)"
          maxLength={64}
          autoComplete="new-password"
        />
        <button type="submit" disabled={creating}>
          {creating ? '...' : '+'}
        </button>
      </form>
      {error && <p className="channel-error">{error}</p>}
    </nav>
  );
}
