/* eslint-disable react-refresh/only-export-components */
// Context files intentionally export both a Provider component and a hook;
// react-refresh would warn about mixed exports — suppressed here by convention.
import {
  createContext,
  useContext,
  useEffect,
  useState,
  type ReactNode,
} from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  apiClient,
  setActiveProfileId as syncApiHeader,
} from '../lib/apiClient';

interface ActiveProfileContextValue {
  activeProfileId: string | null;
  /** Updates the context state and synchronises the axios header immediately. */
  setActiveProfileId: (id: string | null) => void;
  /**
   * Switches the active profile server-side via `PUT /api/active-profile`,
   * then updates local state/header and invalidates every profile-scoped
   * query so the whole app re-fetches under the new profile.
   */
  switchActiveProfile: (id: string) => Promise<void>;
  isSwitching: boolean;
}

interface ActiveProfileDto {
  id: string;
}

const ActiveProfileContext = createContext<ActiveProfileContextValue | null>(
  null,
);

function ActiveProfileProvider({ children }: { children: ReactNode }) {
  const [activeProfileId, setIdState] = useState<string | null>(null);
  const queryClient = useQueryClient();

  function setActiveProfileId(id: string | null): void {
    setIdState(id);
    syncApiHeader(id);
  }

  // Every server-scoped React Query cache uses activeProfileId as a key
  // segment (see useJobs, useProfiles, useDashboard*, etc.), so a plain
  // `invalidateQueries()` with no key re-fetches all of them under the new
  // profile — this is what "switching profile re-scopes all server state" means.
  const switchMutation = useMutation({
    mutationFn: async (id: string) => {
      await apiClient.put('/active-profile', { profile_id: id });
      return id;
    },
    onSuccess: (id) => {
      setActiveProfileId(id);
      void queryClient.invalidateQueries();
    },
  });

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
    <ActiveProfileContext.Provider
      value={{
        activeProfileId,
        setActiveProfileId,
        switchActiveProfile: async (id) => {
          await switchMutation.mutateAsync(id);
        },
        isSwitching: switchMutation.isPending,
      }}
    >
      {children}
    </ActiveProfileContext.Provider>
  );
}

function useActiveProfile(): ActiveProfileContextValue {
  const ctx = useContext(ActiveProfileContext);
  if (ctx === null) {
    throw new Error(
      'useActiveProfile must be used inside <ActiveProfileProvider>',
    );
  }
  return ctx;
}

export { ActiveProfileProvider, useActiveProfile };
