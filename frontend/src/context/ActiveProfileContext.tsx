/* eslint-disable react-refresh/only-export-components */
// Context files intentionally export both a Provider component and a hook;
// react-refresh would warn about mixed exports — suppressed here by convention.
import { createContext, useContext, useState, type ReactNode } from 'react';
import { setActiveProfileId as syncApiHeader } from '../lib/apiClient';

interface ActiveProfileContextValue {
  activeProfileId: string | null;
  /** Updates the context state and synchronises the axios header immediately. */
  setActiveProfileId: (id: string | null) => void;
}

const ActiveProfileContext = createContext<ActiveProfileContextValue | null>(null);

function ActiveProfileProvider({ children }: { children: ReactNode }) {
  const [activeProfileId, setIdState] = useState<string | null>(null);

  function setActiveProfileId(id: string | null): void {
    setIdState(id);
    syncApiHeader(id);
  }

  return (
    <ActiveProfileContext.Provider value={{ activeProfileId, setActiveProfileId }}>
      {children}
    </ActiveProfileContext.Provider>
  );
}

function useActiveProfile(): ActiveProfileContextValue {
  const ctx = useContext(ActiveProfileContext);
  if (ctx === null) {
    throw new Error('useActiveProfile must be used inside <ActiveProfileProvider>');
  }
  return ctx;
}

export { ActiveProfileProvider, useActiveProfile };
