import { API_HOST } from './host';
const BASE = `${API_HOST}/api/v1/auth`;

interface TokenPair {
  user_id: string;
  access_token: string;
  refresh_token: string;
  expires_in: number;
}

interface User {
  id: string;
  username: string;
  email: string;
  display_name: string;
  roles: string[];
}

async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const res = await fetch(`${BASE}${path}`, {
    headers: { 'Content-Type': 'application/json', ...options.headers },
    ...options,
  });
  const body = await res.json();
  if (!res.ok) {
    const msg = body?.error?.message ?? res.statusText;
    throw new Error(msg);
  }
  return body as T;
}

export function register(username: string, email: string, password: string) {
  return request<TokenPair>('/register', {
    method: 'POST',
    body: JSON.stringify({ username, email, password }),
  });
}

export function login(email: string, password: string) {
  return request<TokenPair>('/login', {
    method: 'POST',
    body: JSON.stringify({ email, password }),
  });
}

export function getMe(token: string) {
  return request<User>('/me', {
    headers: { Authorization: `Bearer ${token}` },
  });
}

export function logout(accessToken: string, refreshToken: string) {
  return request<{ message: string }>('/logout', {
    method: 'POST',
    headers: { Authorization: `Bearer ${accessToken}` },
    body: JSON.stringify({ refresh_token: refreshToken }),
  });
}

export function refreshTokens(refreshToken: string) {
  return request<TokenPair>('/refresh', {
    method: 'POST',
    body: JSON.stringify({ refresh_token: refreshToken }),
  });
}
