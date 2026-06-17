import { useEffect, useRef } from 'react';
import { useChannelStore, type Member } from '../stores/channelStore';
import { onWS } from '../api/ws';

function MemberItem({ member }: { member: Member }) {
  const name = member.display_name ?? member.user_id.slice(0, 8);
  return (
    <div className={`member-item ${member.speaking ? 'speaking' : ''}`}>
      <span className="member-indicator" />
      {member.speaking ? (
        <span className="member-name overprint" data-text={name}>
          {name}
        </span>
      ) : (
        <span className="member-name">{name}</span>
      )}
      <span className="member-badges">
        {member.muted && <span className="badge muted" title="Muted">M</span>}
      </span>
    </div>
  );
}

export function MemberPanel() {
  const members = useChannelStore((s) => s.members);
  const currentId = useChannelStore((s) => s.currentChannelId);
  const setMembers = useChannelStore((s) => s.setMembers);

  // Use ref to avoid stale closure — currentId at callback execution time, not registration time
  const currentIdRef = useRef(currentId);
  currentIdRef.current = currentId;

  useEffect(() => {
    const unsub1 = onWS('channel_joined', (msg: any) => {
      const p = msg.payload;
      if (!p || p.channel_id !== currentIdRef.current) return;
      if (p.members && Array.isArray(p.members)) {
        // Deduplicate by user_id
        const seen = new Set<string>();
        const deduped = p.members.filter((m: Member) => {
          if (seen.has(m.user_id)) return false;
          seen.add(m.user_id);
          return true;
        });
        setMembers(deduped);
      }
    });
    const unsub2 = onWS('presence_update', (msg: any) => {
      const p = msg.payload;
      if (!p || p.channel_id !== currentIdRef.current) return;
      if (p.members && Array.isArray(p.members)) {
        const seen = new Set<string>();
        const deduped = p.members.filter((m: Member) => {
          if (seen.has(m.user_id)) return false;
          seen.add(m.user_id);
          return true;
        });
        setMembers(deduped);
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

  // Deduplicate at render time too
  const seen = new Set<string>();
  const uniqueMembers = members.filter((m) => {
    if (seen.has(m.user_id)) return false;
    seen.add(m.user_id);
    return true;
  });

  return (
    <div className="member-panel">
      <div className="member-panel-header">
        <h3>Members</h3>
        <span className="member-count">{uniqueMembers.length}</span>
      </div>
      <div className="member-list">
        {uniqueMembers.map((m) => (
          <MemberItem key={m.user_id} member={m} />
        ))}
      </div>
    </div>
  );
}
