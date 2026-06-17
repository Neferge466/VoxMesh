import { useState, useEffect, useCallback, useRef } from 'react';
import { useChannelStore } from '../stores/channelStore';
import { useAuthStore } from '../stores/authStore';
import { sendWS, onWS } from '../api/ws';
import { uuid } from '../lib/uuid';

interface Message {
  sender: string;
  content: string;
  time: string;
}

function storageKey(channelId: string) { return `voxmesh-msgs-${channelId}`; }

export function ChatBox() {
  const currentId = useChannelStore((s) => s.currentChannelId);
  const user = useAuthStore((s) => s.user);
  const [messages, setMessages] = useState<Message[]>([]);
  const [input, setInput] = useState('');
  const bottomRef = useRef<HTMLDivElement>(null);

  // Load messages from localStorage when channel changes
  useEffect(() => {
    if (currentId) {
      try {
        const raw = localStorage.getItem(storageKey(currentId));
        setMessages(raw ? JSON.parse(raw) : []);
      } catch { setMessages([]); }
    }
  }, [currentId]);

  // Listen for incoming WebSocket chat messages
  useEffect(() => {
    return onWS('chat_message', (msg: any) => {
      const p = msg.payload;
      if (!p || p.channel_id !== currentId) return;
      // Skip messages from self — already shown via optimistic update in send()
      if (user && p.sender_id === user.id) return;
      const incoming: Message = {
        sender: p.sender_name ?? p.sender ?? 'unknown',
        content: p.content ?? p.text ?? '',
        time: new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }),
      };
      setMessages((prev) => {
        const next = [...prev, incoming];
        if (currentId) {
          try { localStorage.setItem(storageKey(currentId), JSON.stringify(next.slice(-200))); } catch { /* quota */ }
        }
        return next;
      });
    });
  }, [currentId, user]);

  // Scroll to bottom on new messages
  useEffect(() => { bottomRef.current?.scrollIntoView({ behavior: 'smooth' }); }, [messages]);

  const send = useCallback(() => {
    if (!input.trim() || !currentId || !user) return;
    const content = input.trim();
    const msg: Message = {
      sender: user.display_name ?? user.username,
      content,
      time: new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }),
    };
    // Local optimistic update
    const next = [...messages, msg];
    setMessages(next);
    try { localStorage.setItem(storageKey(currentId), JSON.stringify(next.slice(-200))); } catch { /* quota */ }
    setInput('');
    // Send via WebSocket for other clients
    sendWS({
      type: 'chat_message',
      id: uuid(),
      timestamp_ms: Date.now(),
      payload: { channel_id: currentId, sender: msg.sender, content },
    });
  }, [input, currentId, user, messages]);

  if (!currentId) return null;

  return (
    <div className="chat-box">
      <div className="chat-messages">
        {messages.length === 0 && <p className="chat-empty">No messages yet</p>}
        {messages.map((m, i) => {
          const isSelf = user && (m.sender === user.display_name || m.sender === user.username);
          return (
            <div key={`${m.sender}-${m.time}-${i}`} className={`chat-msg ${isSelf ? 'chat-self' : ''}`}>
              <span className="msg-sender">{m.sender}:</span>
              <span className="msg-content">{m.content}</span>
              <span className="msg-time">{m.time}</span>
            </div>
          );
        })}
        <div ref={bottomRef} />
      </div>
      <form className="chat-input" onSubmit={(e) => { e.preventDefault(); send(); }}>
        <input value={input} onChange={(e) => setInput(e.target.value)} placeholder="Type a message..." />
        <button type="submit">Send</button>
      </form>
    </div>
  );
}
