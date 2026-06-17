// Auto-detect server host from window.location when no env var is set.
// The same build works for localhost (dev) and LAN IP (external testing).
export const API_HOST =
  import.meta.env.VITE_API_BASE ||
  `${window.location.protocol}//${window.location.hostname}:8085`;

export const WS_HOST =
  import.meta.env.VITE_WS_URL ||
  `ws://${window.location.hostname}:8085/ws`;
