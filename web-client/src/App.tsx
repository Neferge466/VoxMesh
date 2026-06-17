import { useState, useEffect, useRef, type FormEvent } from 'react';
import { Layout } from './components/Layout';
import { ChannelTree } from './components/ChannelTree';
import { MemberPanel } from './components/MemberPanel';
import { AudioControls } from './components/AudioControls';
import { ChatBox } from './components/ChatBox';
import { RemoteAudio } from './components/RemoteAudio';
import { ThemeToggle } from './components/ThemeToggle';
import { useChannelStore } from './stores/channelStore';
import { useAuthStore } from './stores/authStore';
import * as authApi from './api/auth';
import * as channelApi from './api/channels';
import { connectWS, disconnectWS } from './api/ws';
import './App.css';

function MainContent() {
  const currentId = useChannelStore((s) => s.currentChannelId);

  if (!currentId) {
    return (
      <div className="main-content">
        <div className="main-welcome">
          <h2 className="welcome-headline">
            Welcome to <span className="welcome-accent">VoxMesh</span>
          </h2>
          <p className="welcome-sub">
            Cross-device voice communication system.
            Join a channel to start speaking.
          </p>
          <div className="welcome-actions">
            <button
              className="welcome-btn welcome-btn-primary"
              onClick={() => {
                const input = document.getElementById('channel-create-input') as HTMLInputElement | null;
                input?.focus();
                input?.scrollIntoView({ behavior: 'smooth' });
              }}
            >
              + Create Channel
            </button>
            <span className="welcome-hint">or select a channel from the sidebar</span>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="main-content">
      <ChatBox />
      <AudioControls />
      <RemoteAudio />
    </div>
  );
}

function LoginForm() {
  const [mode, setMode] = useState<'login' | 'register'>('login');
  const [username, setUsername] = useState('');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const authLogin = useAuthStore((s) => s.login);

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setError('');
    setLoading(true);
    try {
      if (mode === 'register') {
        const pair = await authApi.register(username, email, password);
        const user = await authApi.getMe(pair.access_token);
        authLogin(
          { id: user.id, username: user.username, email: user.email, display_name: user.display_name, roles: user.roles },
          pair.access_token,
          pair.refresh_token,
        );
      } else {
        const pair = await authApi.login(email, password);
        const user = await authApi.getMe(pair.access_token);
        authLogin(
          { id: user.id, username: user.username, email: user.email, display_name: user.display_name, roles: user.roles },
          pair.access_token,
          pair.refresh_token,
        );
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Request failed');
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="auth-container">
      <form className="auth-form" onSubmit={handleSubmit}>
        <div className="auth-form-header">
          <h2 className="auth-title">{mode === 'login' ? 'Sign In' : 'Create Account'}</h2>
          <ThemeToggle />
        </div>

        {error && <div className="auth-error">{error}</div>}

        {mode === 'register' && (
          <label className="auth-field">
            <span>Username</span>
            <input
              type="text"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              required
              minLength={3}
              maxLength={32}
              autoComplete="username"
            />
          </label>
        )}

        <label className="auth-field">
          <span>Email</span>
          <input
            type="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            required
            autoComplete="email"
          />
        </label>

        <label className="auth-field">
          <span>Password</span>
          <input
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            required
            minLength={6}
            autoComplete={mode === 'login' ? 'current-password' : 'new-password'}
          />
        </label>

        <button className="auth-submit" type="submit" disabled={loading}>
          {loading ? 'Please wait...' : mode === 'login' ? 'Sign In' : 'Create Account'}
        </button>

        <button
          type="button"
          className="auth-switch"
          onClick={() => { setMode(mode === 'login' ? 'register' : 'login'); setError(''); }}
        >
          {mode === 'login' ? 'Need an account? Register' : 'Already have an account? Sign in'}
        </button>
      </form>
    </div>
  );
}

export default function App() {
  const user = useAuthStore((s) => s.user);
  const accessToken = useAuthStore((s) => s.accessToken);
  const refreshToken = useAuthStore((s) => s.refreshToken);
  const logout = useAuthStore((s) => s.logout);
  const setChannels = useChannelStore((s) => s.setChannels);

  function handleLogout() {
    if (accessToken && refreshToken) authApi.logout(accessToken, refreshToken).catch(() => {});
    logout();
  }

  const [channelError, setChannelError] = useState('');

  useEffect(() => {
    if (!accessToken) return;
    channelApi.listChannels(accessToken).then((list) => {
      setChannels(list);
      setChannelError('');
    }).catch((err) => {
      const msg = err instanceof Error ? err.message : 'Failed to load channels';
      // Token expired or invalid — force re-login
      if (msg.toLowerCase().includes('token') || msg.toLowerCase().includes('expired') || msg.toLowerCase().includes('unauthorized')) {
        logout();
        return;
      }
      setChannelError(msg);
    });
  }, [accessToken, setChannels, logout]);

  // WebSocket connection (StrictMode-safe: only disconnect on real token change, not re-mount)
  const wsTokenRef = useRef<string | null>(null);
  useEffect(() => {
    if (!accessToken) {
      if (wsTokenRef.current !== null) { disconnectWS(); wsTokenRef.current = null; }
      return;
    }
    if (wsTokenRef.current === accessToken) return;
    wsTokenRef.current = accessToken;
    connectWS(`${import.meta.env.VITE_WS_URL || `ws://${window.location.hostname}:8085/ws`}?token=${accessToken}`);
    return () => {
      // Only disconnect if token actually changed, not on StrictMode re-mount
      if (wsTokenRef.current !== accessToken) disconnectWS();
    };
  }, [accessToken]);

  // Token refresh: refresh 5 minutes before expiry (access token lifetime is 1 hour)
  useEffect(() => {
    if (!accessToken || !refreshToken) return;
    const interval = setInterval(() => {
      authApi.refreshTokens(refreshToken).then((pair) => {
        useAuthStore.getState().login(
          useAuthStore.getState().user!,
          pair.access_token,
          pair.refresh_token,
        );
      }).catch(() => {
        // Refresh failed — user will get 401 on next API call and can re-login
      });
    }, 55 * 60 * 1000);
    return () => clearInterval(interval);
  }, [accessToken, refreshToken]);

  if (!user) {
    return <LoginForm />;
  }

  return (
    <Layout
      sidebar={<>
        <ChannelTree token={accessToken!} />
        {channelError && <div className="channel-error-banner">{channelError}</div>}
      </>}
      headerRight={
        <div className="header-actions">
          <span className="header-user">{user.display_name ?? user.username}</span>
          <ThemeToggle />
          <button className="header-logout" onClick={handleLogout}>Logout</button>
        </div>
      }
    >
      <MainContent />
      <MemberPanel />
    </Layout>
  );
}
