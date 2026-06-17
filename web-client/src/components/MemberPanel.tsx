import { useEffect, useRef, useState } from 'react';
import { useChannelStore, type Member } from '../stores/channelStore';
import { useAuthStore } from '../stores/authStore';
import { useWebRTCStore } from '../stores/webrtcStore';
import { onWS } from '../api/ws';
import { kickUser } from '../api/channels';

function MemberItem({ member, isSelf, token, channelId, onKicked }: {
  member: Member;
  isSelf: boolean;
  token: string;
  channelId: string;
  onKicked: () => void;
}) {
  const [kicking, setKicking] = useState(false);
  const [hover, setHover] = useState(false);
  const name = member.display_name ?? member.user_id.slice(0, 8);
  const volume = useWebRTCStore((s) => s.volumes.get(member.user_id) ?? 1.0);
  const setVolume = useWebRTCStore((s) => s.setVolume);

  const handleKick = async () => {
    if (kicking) return;
    setKicking(true);
    try {
      await kickUser(token, channelId, member.user_id);
      onKicked();
    } catch (e) {
      console.error('[member] kick failed:', e);
    } finally {
      setKicking(false);
    }
  };

  const stateClass = member.speaking ? 'speaking' : member.muted ? 'muted' : 'idle';

  return (
    <div
      className={`member-item ${stateClass} ${isSelf ? 'is-self' : ''}`}
      onMouseEnter={() => setHover(true)}
      onMouseLeave={() => setHover(false)}
    >
      <div className="member-row">
        <span className="member-indicator" />
        <span className="member-name" data-text={name}>
          {member.speaking ? <span className="overprint">{name}</span> : name}
        </span>
        <span className="member-meta">
          {isSelf && <span className="tag tag-self">you</span>}
          {member.muted && <span className="tag tag-muted">muted</span>}
          {member.client_type === 'embedded' && <span className="tag tag-hw">hw</span>}
        </span>
        {!isSelf && hover && (
          <button
            className="member-kick"
            title="Kick from channel"
            onClick={handleKick}
            disabled={kicking}
          >
            {kicking ? '…' : '×'}
          </button>
        )}
      </div>
      {!isSelf && hover && (
        <div className="member-volume-row">
          <input
            type="range"
            className="member-volume"
            min="0"
            max="1"
            step="0.05"
            value={volume}
            onChange={(e) => setVolume(member.user_id, parseFloat(e.target.value))}
          />
          <span className="volume-label">{Math.round(volume * 100)}</span>
        </div>
      )}
    </div>
  );
}

export function MemberPanel() {
  const members = useChannelStore((s) => s.members);
  const currentId = useChannelStore((s) => s.currentChannelId);
  const setMembers = useChannelStore((s) => s.setMembers);
  const userId = useAuthStore((s) => s.user?.id);
  const token = useAuthStore((s) => s.token);

  const currentIdRef = useRef(currentId);
  currentIdRef.current = currentId;

  useEffect(() => {
    const unsub1 = onWS('channel_joined', (msg: any) => {
      const p = msg.payload;
      if (!p || p.channel_id !== currentIdRef.current) return;
      if (p.members && Array.isArray(p.members)) {
        const seen = new Set<string>();
        setMembers(p.members.filter((m: Member) => {
          if (seen.has(m.user_id)) return false;
          seen.add(m.user_id);
          return true;
        }));
      }
    });
    const unsub2 = onWS('presence_update', (msg: any) => {
      const p = msg.payload;
      if (!p || p.channel_id !== currentIdRef.current) return;
      if (p.members && Array.isArray(p.members)) {
        const seen = new Set<string>();
        setMembers(p.members.filter((m: Member) => {
          if (seen.has(m.user_id)) return false;
          seen.add(m.user_id);
          return true;
        }));
      }
    });
    const unsub3 = onWS('user_speaking', (msg: any) => {
      const p = msg.payload;
      if (!p) return;
      setMembers((prev: Member[]) =>
        prev.map((m) => (m.user_id === p.user_id ? { ...m, speaking: p.speaking } : m))
      );
    });
    return () => { unsub1(); unsub2(); unsub3(); };
  }, [currentId, setMembers]);

  if (!currentId) return null;

  const seen = new Set<string>();
  const uniqueMembers = members.filter((m) => {
    if (seen.has(m.user_id)) return false;
    seen.add(m.user_id);
    return true;
  });

  const sorted = [...uniqueMembers].sort((a, b) => {
    if (a.user_id === userId && b.user_id !== userId) return -1;
    if (b.user_id === userId && a.user_id !== userId) return 1;
    if (a.speaking && !b.speaking) return -1;
    if (b.speaking && !a.speaking) return 1;
    const nameA = (a.display_name ?? a.user_id).toLowerCase();
    const nameB = (b.display_name ?? b.user_id).toLowerCase();
    return nameA.localeCompare(nameB);
  });

  return (
    <aside className="member-panel" aria-label="Channel members">
      <div className="member-panel-header">
        <h3 className="member-panel-title">Members</h3>
        <span className="member-count">{sorted.length}</span>
      </div>
      <div className="member-list">
        {sorted.map((m) => (
          <MemberItem
            key={m.user_id}
            member={m}
            isSelf={m.user_id === userId}
            token={token ?? ''}
            channelId={currentId}
            onKicked={() => {}}
          />
        ))}
      </div>
    </aside>
  );
}
