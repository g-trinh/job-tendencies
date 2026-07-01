/* eslint-disable react-refresh/only-export-components */
// Context files intentionally export both a Provider component and a hook;
// react-refresh would warn about mixed exports — suppressed here by convention.
import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useState,
  type ReactNode,
} from 'react';
import { apiClient, setOnUnauthorized } from '../lib/apiClient';

/** Shape of the user object returned by GET /api/auth/me. */
export interface AuthUser {
  email: string;
}

type AuthStatus = 'loading' | 'authenticated' | 'unauthenticated';

interface AuthContextValue {
  status: AuthStatus;
  user: AuthUser | null;
  /** Re-fetches /api/auth/me. Call this after a successful login. */
  checkAuth: () => Promise<void>;
  /** Posts to /api/auth/logout, then clears local auth state. */
  logout: () => Promise<void>;
}

const AuthContext = createContext<AuthContextValue | null>(null);

function AuthProvider({ children }: { children: ReactNode }) {
  const [status, setStatus] = useState<AuthStatus>('loading');
  const [user, setUser] = useState<AuthUser | null>(null);

  const checkAuth = useCallback(async () => {
    try {
      const { data } = await apiClient.get<AuthUser>('/auth/me');
      setUser(data);
      setStatus('authenticated');
    } catch {
      setUser(null);
      setStatus('unauthenticated');
    }
  }, []);

  useEffect(() => {
    // Register the global 401 handler so that any guarded /api call that returns
    // 401 (e.g. an expired session mid-use) drops the user back to the login screen.
    setOnUnauthorized(() => {
      setUser(null);
      setStatus('unauthenticated');
    });

    // Stub: calls GET /api/auth/me; backend P4-BE-2 must implement this endpoint.
    // Expected contract: 200 { email: string } when the session cookie is valid,
    // 401 (no body required) when not.
    checkAuth();

    return () => {
      setOnUnauthorized(null);
    };
  }, [checkAuth]);

  const logout = useCallback(async () => {
    try {
      // Backend P4-BE-2 must implement POST /api/auth/logout.
      // Expected contract: 200 (or 204) + clears the httpOnly session cookie.
      await apiClient.post('/auth/logout');
    } finally {
      setUser(null);
      setStatus('unauthenticated');
    }
  }, []);

  return (
    <AuthContext.Provider value={{ status, user, checkAuth, logout }}>
      {children}
    </AuthContext.Provider>
  );
}

function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext);
  if (ctx === null) {
    throw new Error('useAuth must be used inside <AuthProvider>');
  }
  return ctx;
}

export { AuthProvider, useAuth };
