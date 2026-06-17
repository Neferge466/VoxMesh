import { create } from 'zustand';

interface User {
  id: string;
  username: string;
  email: string;
  display_name: string;
  roles: string[];
}

interface AuthState {
  user: User | null;
  accessToken: string | null;
  refreshToken: string | null;
  login: (user: User, access: string, refresh: string) => void;
  logout: () => void;
}

function isTokenExpired(token: string): boolean {
  try {
    const parts = token.split('.');
    if (parts.length !== 3) return true;
    const payload = JSON.parse(atob(parts[1]));
    const exp = payload.exp as number;
    const now = Math.floor(Date.now() / 1000);
    return exp < now + 30; // 30s leeway
  } catch {
    return true;
  }
}

function loadPersisted(): Partial<AuthState> {
  try {
    const raw = localStorage.getItem('voxmesh-auth');
    if (!raw) return {};
    const data = JSON.parse(raw);
    const token = data.accessToken ?? null;
    // Discard expired tokens so the user sees the login form immediately
    if (token && isTokenExpired(token)) {
      localStorage.removeItem('voxmesh-auth');
      return {};
    }
    return {
      user: data.user ?? null,
      accessToken: token,
      refreshToken: data.refreshToken ?? null,
    };
  } catch {
    return {};
  }
}

function persist(state: AuthState) {
  try {
    localStorage.setItem('voxmesh-auth', JSON.stringify({
      user: state.user,
      accessToken: state.accessToken,
      refreshToken: state.refreshToken,
    }));
  } catch { /* quota exceeded — ignore */ }
}

function clearPersisted() {
  try {
    localStorage.removeItem('voxmesh-auth');
  } catch { /* ignore */ }
}

const initial = loadPersisted();

export const useAuthStore = create<AuthState>((set) => ({
  user: initial.user ?? null,
  accessToken: initial.accessToken ?? null,
  refreshToken: initial.refreshToken ?? null,
  login: (user, accessToken, refreshToken) => {
    const next = { user, accessToken, refreshToken };
    persist(next as AuthState);
    set(next);
  },
  logout: () => {
    clearPersisted();
    set({ user: null, accessToken: null, refreshToken: null });
  },
}));
