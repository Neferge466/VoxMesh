import { type ReactNode, useState, useCallback, useEffect } from 'react';
import { ThemeToggle } from './ThemeToggle';
import { getWSState, onWSStateChange, type WSState } from '../api/ws';

interface Props {
  children: ReactNode;
  sidebar?: ReactNode;
  headerRight?: ReactNode;
}

export function Layout({ children, sidebar, headerRight }: Props) {
  const [sidebarOpen, setSidebarOpen] = useState(false);
  const [wsState, setWsState] = useState<WSState>(getWSState);
  const closeSidebar = useCallback(() => setSidebarOpen(false), []);

  useEffect(() => {
    return onWSStateChange(setWsState);
  }, []);

  const stateLabel = wsState === 'connected' ? 'Connected' : wsState === 'connecting' ? 'Connecting…' : 'Offline';

  return (
    <div className="app-shell">
      <header className="app-header">
        <div className="header-brand">
          {sidebar && (
            <button
              className="hamburger"
              onClick={() => setSidebarOpen((v) => !v)}
              aria-label={sidebarOpen ? 'Close menu' : 'Open menu'}
            >
              <span className={sidebarOpen ? 'hamburger-line open' : 'hamburger-line'} />
            </button>
          )}
          <span className="header-logo">VX</span>
          <h1 className="header-title">VoxMesh</h1>
          <span className={`ws-status ws-${wsState}`} title={stateLabel} />
        </div>
        <div className="header-actions">
          {headerRight ?? <ThemeToggle />}
        </div>
      </header>
      <div className="app-body">
        {sidebarOpen && <div className="sidebar-overlay" onClick={closeSidebar} />}
        {sidebar && (
          <aside className={`app-sidebar${sidebarOpen ? ' open' : ''}`}>
            <div className="sidebar-close-row">
              <button className="sidebar-close-btn" onClick={closeSidebar} aria-label="Close">
                &times;
              </button>
            </div>
            {sidebar}
          </aside>
        )}
        <main className="app-main" onClick={sidebarOpen ? closeSidebar : undefined}>
          {children}
        </main>
      </div>
    </div>
  );
}
