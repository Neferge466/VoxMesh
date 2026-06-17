import { type ReactNode, useState, useCallback } from 'react';
import { ThemeToggle } from './ThemeToggle';

interface Props {
  children: ReactNode;
  sidebar?: ReactNode;
  headerRight?: ReactNode;
}

export function Layout({ children, sidebar, headerRight }: Props) {
  const [sidebarOpen, setSidebarOpen] = useState(false);
  const closeSidebar = useCallback(() => setSidebarOpen(false), []);

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
