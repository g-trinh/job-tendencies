/* eslint-disable react-refresh/only-export-components */
// Context files intentionally export both a Provider component and a hook;
// react-refresh would warn about mixed exports — suppressed here by convention.
import { createContext, useContext, useEffect, useState, type ReactNode } from 'react';
import { useQuery } from '@tanstack/react-query';
import { apiClient, setActiveProfileId as syncApiHeader } from '../lib/apiClient';

interface ActiveProfileContextValue {
  activeProfileId: string | null;
  /** Updates the context state and synchronises the axios header immediately. */
  setActiveProfileId: (id: string | null) => void;
}

interface ActiveProfileDto {
  id: string;
}

const ActiveProfileContext = createContext<ActiveProfileContextValue | null>(null);

function ActiveProfileProvider({ children }: { children: ReactNode }) {
  const [activeProfileId, setIdState] = useState<string | null>(null);

  function setActiveProfileId(id: string | null): void {
    setIdState(id);
    syncApiHeader(id);
  }

  // Bootstrap the active profile once on mount. The id is needed before any
  // scoped query (jobs, dashboard) can run, since it backs the X-Active-Profile
  // header and every scoped React Query cache key.
  const { data: bootstrapped } = useQuery({
    queryKey: ['active-profile'],
    queryFn: async () => {
      const { data } = await apiClient.get<ActiveProfileDto>('/active-profile');
      return data.id;
    },
  });

  useEffect(() => {
    if (bootstrapped !== undefined && bootstrapped !== activeProfileId) {
      setActiveProfileId(bootstrapped);
    }
  }, [bootstrapped, activeProfileId]);

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
