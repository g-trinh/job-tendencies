import type { ReactNode } from 'react';
import { NavLink } from 'react-router-dom';
import { ProfileSwitcher } from '../features/profiles';

interface AppShellProps {
  children: ReactNode;
}

const NAV_LINKS: { to: string; label: string }[] = [
  { to: '/dashboard', label: 'Tableau de bord' },
  { to: '/', label: 'Offres' },
  { to: '/kanban', label: 'Candidatures' },
  { to: '/profiles', label: 'Profils' },
  { to: '/boards', label: 'Sources' },
  { to: '/contacts', label: 'Contacts' },
  { to: '/pipeline', label: 'Pipeline' },
];

/**
 * Shared app shell (sidebar + topbar + page) ported from `template/` (design
 * system doc: design_changes.md). One layout component replaces the
 * per-screen `.app`/`.app__sidebar`/`.topbar`/`.page` markup duplicated in
 * the static reference — every routed page renders inside `.page`.
 */
function AppShell({ children }: AppShellProps) {
  return (
    <div className="app">
      <aside className="app__sidebar">
        <span className="app__brand">
          <span className="app__brand-mark">JT</span> Job Tendencies
        </span>
        <nav className="nav" aria-label="Navigation principale">
          <span className="nav__section">Écrans</span>
          {NAV_LINKS.map((link) => (
            <NavLink
              key={link.to}
              to={link.to}
              end={link.to === '/'}
              className="nav__link"
            >
              {link.label}
            </NavLink>
          ))}
        </nav>
      </aside>

      <div className="app__main">
        <header className="topbar">
          <ProfileSwitcher />
        </header>
        <div className="page">{children}</div>
      </div>
    </div>
  );
}

export { AppShell };
