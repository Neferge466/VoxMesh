import { uuid } from '../lib/uuid';

type Listener = (data: unknown) => void;
type BinaryListener = (data: ArrayBuffer) => void;

export type WSState = 'connecting' | 'connected' | 'disconnected';

const listeners = new Map<string, Set<Listener>>();
const binaryListeners = new Set<BinaryListener>();
const pendingMessages: object[] = [];
const stateListeners = new Set<(state: WSState) => void>();

let ws: WebSocket | null = null;
let reconnectTimer: ReturnType<typeof setTimeout> | null = null;
let reconnectCount = 0;
let wsUrl = '';
let stopped = false;
let lastChannelId: string | null = null;
let _wsState: WSState = 'disconnected';

function setWSState(state: WSState) {
  if (_wsState === state) return;
  _wsState = state;
  stateListeners.forEach((fn) => fn(state));
}

export function getWSState(): WSState {
  return _wsState;
}

export function onWSStateChange(fn: (state: WSState) => void): () => void {
  stateListeners.add(fn);
  return () => { stateListeners.delete(fn); };
}

export function connectWS(url: string) {
  stopped = false;
  if (wsUrl === url && ws && (ws.readyState === WebSocket.OPEN || ws.readyState === WebSocket.CONNECTING)) return;
  wsUrl = url;
  doConnect();
}

export function setActiveChannel(channelId: string | null) {
  lastChannelId = channelId;
  if (channelId && ws?.readyState === WebSocket.OPEN) {
    // If already connected, send join immediately
    sendWS({ type: 'join_channel', id: uuid(), timestamp_ms: Date.now(), payload: { channel_id: channelId } });
  }
}

function doConnect() {
  if (!wsUrl || stopped) return;
  if (reconnectTimer) { clearTimeout(reconnectTimer); reconnectTimer = null; }

  if (ws) {
    try { ws.onclose = null; ws.onerror = null; ws.close(); } catch { /* ignore */ }
    ws = null;
  }

  setWSState('connecting');
  ws = new WebSocket(wsUrl);

  ws.onopen = () => {
    console.log('[ws] open, pending=%d, channel=%s', pendingMessages.length, lastChannelId);
    setWSState('connected');
    reconnectCount = 0;
    if (lastChannelId) {
      pendingMessages.unshift({ type: 'join_channel', id: uuid(), timestamp_ms: Date.now(), payload: { channel_id: lastChannelId } });
    }
    while (pendingMessages.length > 0) {
      const msg = pendingMessages.shift()!;
      try { ws?.send(JSON.stringify(msg)); } catch { break; }
    }
  };

  ws.onmessage = (event) => {
    if (event.data instanceof ArrayBuffer) {
      binaryListeners.forEach((fn) => fn(event.data));
      return;
    }
    if (event.data instanceof Blob) {
      event.data.arrayBuffer().then((buf) => binaryListeners.forEach((fn) => fn(buf)));
      return;
    }
    try {
      const msg = JSON.parse(event.data as string);
      const type = msg.type as string;
      if (type && listeners.has(type)) {
        listeners.get(type)!.forEach((fn) => fn(msg));
      }
    } catch { /* ignore */ }
  };

  ws.onerror = () => { /* browser logs this, we handle via onclose */ };

  ws.onclose = () => {
    ws = null;
    setWSState('disconnected');
    if (stopped || !wsUrl) return;
    const delay = Math.min(2000 * Math.pow(2, reconnectCount), 30000);
    reconnectCount++;
    reconnectTimer = setTimeout(doConnect, delay);
  };
}

export function disconnectWS() {
  stopped = true;
  if (reconnectTimer) { clearTimeout(reconnectTimer); reconnectTimer = null; }
  if (ws) { try { ws.onclose = null; ws.onerror = null; ws.close(); } catch { /* ignore */ } }
  ws = null;
  wsUrl = '';
  lastChannelId = null;
  reconnectCount = 0;
  setWSState('disconnected');
}

export function sendWS(msg: object) {
  if (ws?.readyState === WebSocket.OPEN) {
    try { ws.send(JSON.stringify(msg)); } catch { pendingMessages.push(msg); }
  } else {
    pendingMessages.push(msg);
    if (pendingMessages.length > 50) pendingMessages.shift();
  }
}

export function sendBinaryWS(data: ArrayBuffer) {
  if (ws?.readyState === WebSocket.OPEN) {
    try { ws.send(data); } catch { /* drop */ }
  }
}

export function onWS(type: string, fn: Listener): () => void {
  if (!listeners.has(type)) listeners.set(type, new Set());
  listeners.get(type)!.add(fn);
  return () => { listeners.get(type)?.delete(fn); };
}

export function onBinaryWS(fn: BinaryListener): () => void {
  binaryListeners.add(fn);
  return () => { binaryListeners.delete(fn); };
}
