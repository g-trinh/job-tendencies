import { type ReactNode } from 'react';
import { useAuth } from '../../context/AuthContext';
import { LoginPage } from './LoginPage';

interface RequireAuthProps {
  children: ReactNode;
}

/**
 * App-level route guard. Wraps the entire routed application so that:
 * - While the session check is in-flight, nothing is rendered (avoids flash of content).
 * - When no valid session exists, the French login screen is shown instead of the app.
 * - When a valid session exists, children (the routed app) are rendered normally.
 *
 * The 401 interceptor in apiClient.ts covers mid-session expiry: any guarded
 * /api call that returns 401 triggers AuthContext to set status='unauthenticated',
 * which causes this component to swap back to the login screen.
 */
function RequireAuth({ children }: RequireAuthProps) {
  const { status } = useAuth();

  if (status === 'loading') {
    return null;
  }

  if (status === 'unauthenticated') {
    return <LoginPage />;
  }

  return <>{children}</>;
}

export { RequireAuth };
