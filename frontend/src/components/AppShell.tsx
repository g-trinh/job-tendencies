/* eslint-disable react-refresh/only-export-components */
// This file exports both the AppShell component and the useWidePage hook
// (context files elsewhere follow the same convention) — react-refresh would
// warn about mixed exports, suppressed here by convention.
import {
  Fragment,
  createContext,
  useContext,
  useEffect,
  useState,
  type ReactNode,
} from 'react';
import { NavLink } from 'react-router-dom';
import { ProfileSwitcher } from '../features/profiles';
import { useActiveProfile } from '../context/ActiveProfileContext';
import { useProfiles } from '../features/profiles/useProfiles';
import { useAuth } from '../context/AuthContext';

interface AppShellProps {
  children: ReactNode;
}

interface NavItem {
  to: string;
  label: string;
  end?: boolean;
  icon: ReactNode;
}

interface NavSection {
  title: string;
  items: NavItem[];
}

/** 16px stroke icons copied verbatim from `template/screens/*.html` (§ Shell). */
const NAV_SECTIONS: NavSection[] = [
  {
    title: 'Pilotage',
    items: [
      {
        to: '/dashboard',
        label: 'Tableau de bord',
        icon: (
          <svg viewBox="0 0 24 24" fill="none" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
            <rect x="3" y="3" width="7" height="7" rx="1" />
            <rect x="14" y="3" width="7" height="7" rx="1" />
            <rect x="3" y="14" width="7" height="7" rx="1" />
            <rect x="14" y="14" width="7" height="7" rx="1" />
          </svg>
        ),
      },
      {
        to: '/',
        label: 'Offres',
        end: true,
        icon: (
          <svg viewBox="0 0 24 24" fill="none" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
            <rect x="2.5" y="7" width="19" height="13" rx="2" />
            <path d="M8.5 7V5a2 2 0 0 1 2-2h3a2 2 0 0 1 2 2v2" />
          </svg>
        ),
      },
      {
        to: '/kanban',
        label: 'Candidatures',
        icon: (
          <svg viewBox="0 0 24 24" fill="none" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
            <rect x="3" y="4" width="5" height="16" rx="1" />
            <rect x="10" y="4" width="5" height="10" rx="1" />
            <rect x="17" y="4" width="5" height="7" rx="1" />
          </svg>
        ),
      },
      {
        to: '/pipeline',
        label: 'Pipeline',
        icon: (
          <svg viewBox="0 0 24 24" fill="none" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
            <path d="M22 12h-4l-3 8-6-16-3 8H2" />
          </svg>
        ),
      },
    ],
  },
  {
    title: 'Configuration',
    items: [
      {
        to: '/profiles',
        label: 'Profils',
        icon: (
          <svg viewBox="0 0 24 24" fill="none" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
            <circle cx="12" cy="8" r="3.5" />
            <path d="M5 20.5a7 7 0 0 1 14 0" />
          </svg>
        ),
      },
      {
        to: '/boards',
        label: 'Sources',
        icon: (
          <svg viewBox="0 0 24 24" fill="none" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
            <circle cx="12" cy="12" r="9" />
            <path d="M3 12h18M12 3a14 14 0 0 1 0 18M12 3a14 14 0 0 0 0 18" />
          </svg>
        ),
      },
    ],
  },
  {
    title: 'Réseau',
    items: [
      {
        to: '/contacts',
        label: 'Contacts',
        icon: (
          <svg viewBox="0 0 24 24" fill="none" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
            <circle cx="9" cy="8.5" r="3.5" />
            <path d="M2.5 20a6.5 6.5 0 0 1 13 0M16 5a3.5 3.5 0 0 1 0 7M21.5 20a6.5 6.5 0 0 0-4.2-6" />
          </svg>
        ),
      },
    ],
  },
];

/** Uppercase 1-2 letter initials from an email's local part, for the sidebar avatar. */
function initialsFromEmail(email: string): string {
  const local = email.split('@')[0] ?? '';
  return local.slice(0, 2).toUpperCase() || '?';
}

/** Lets a routed page opt the shell's `.page` container into `.page--wide` (dense table screens). */
const PageWidthContext = createContext<(wide: boolean) => void>(() => {});

/** Call with `true` from a dense table screen; reverts to the default width on unmount. */
function useWidePage(wide: boolean): void {
  const setWide = useContext(PageWidthContext);
  useEffect(() => {
    setWide(wide);
    return () => setWide(false);
  }, [wide, setWide]);
}

/**
 * Shared app shell (sidebar + topbar + page) ported from `template/` (design
 * system doc: design_changes.md, "Polish pass" section). One layout component
 * replaces the per-screen `.app`/`.app__sidebar`/`.topbar`/`.page` markup
 * duplicated in the static reference — every routed page renders inside
 * `<main class="page">`.
 */
function AppShell({ children }: AppShellProps) {
  const { user } = useAuth();
  const { activeProfileId } = useActiveProfile();
  const { data: profiles } = useProfiles();
  const activeProfile = profiles?.find((p) => p.id === activeProfileId);
  const [wide, setWide] = useState(false);

  return (
    <div className="app">
      <aside className="app__sidebar">
        <span className="app__brand">
          <span className="app__brand-mark">JT</span> Job Tendencies
        </span>
        <nav className="nav" aria-label="Navigation principale">
          {NAV_SECTIONS.map((section) => (
            <Fragment key={section.title}>
              <span className="nav__section">{section.title}</span>
              {section.items.map((item) => (
                <NavLink
                  key={item.to}
                  to={item.to}
                  end={item.end}
                  className="nav__link"
                >
                  {item.icon}
                  {item.label}
                </NavLink>
              ))}
            </Fragment>
          ))}
        </nav>
        <div className="app__sidebar-foot">
          <span className="app__avatar" aria-hidden="true">
            {user ? initialsFromEmail(user.email) : '?'}
          </span>
          <span>
            {user?.email ?? 'Utilisateur'}
            {activeProfile && <small>Profil : {activeProfile.name}</small>}
          </span>
        </div>
      </aside>

      <div className="app__main">
        <header className="topbar">
          <ProfileSwitcher />
        </header>
        <main className={wide ? 'page page--wide' : 'page'}>
          <PageWidthContext.Provider value={setWide}>
            {children}
          </PageWidthContext.Provider>
        </main>
      </div>
    </div>
  );
}

export { AppShell, useWidePage };
