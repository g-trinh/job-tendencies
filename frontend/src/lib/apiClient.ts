import axios from 'axios';

/**
 * Module-level active-profile id. Updated by ActiveProfileProvider via
 * setActiveProfileId() below. Kept outside React so it is readable by the
 * axios interceptor without a hook and is trivially testable in isolation.
 */
let _activeProfileId: string | null = null;

/**
 * Module-level unauthorised callback. Set by AuthProvider on mount; cleared on
 * unmount. Invoked by the 401 response interceptor for any non-auth endpoint —
 * i.e. when the session expires mid-use. The callback updates React auth state,
 * which causes RequireAuth to display the login screen.
 */
let _onUnauthorized: (() => void) | null = null;

/** Called by ActiveProfileProvider whenever the active profile changes. */
export function setActiveProfileId(id: string | null): void {
  _activeProfileId = id;
}

/** Called by AuthProvider on mount (set) and unmount (null). */
export function setOnUnauthorized(cb: (() => void) | null): void {
  _onUnauthorized = cb;
}

export const apiClient = axios.create({
  baseURL: '/api',
  // credentials: 'include' equivalent for axios — ensures the httpOnly session
  // cookie is attached to every /api request, even if the origin check changes.
  withCredentials: true,
});

apiClient.interceptors.request.use((config) => {
  if (_activeProfileId !== null) {
    config.headers['X-Active-Profile'] = _activeProfileId;
  }
  return config;
});

/**
 * Auth endpoints manage their own 401 responses (login shows an error,
 * me/logout are handled explicitly in AuthContext). Only redirect the user
 * to the login screen for 401s from guarded application routes.
 */
function isAuthEndpoint(url: string | undefined): boolean {
  return url?.startsWith('/auth/') ?? false;
}

apiClient.interceptors.response.use(
  (response) => response,
  (error) => {
    if (
      axios.isAxiosError(error) &&
      error.response?.status === 401 &&
      !isAuthEndpoint(error.config?.url)
    ) {
      _onUnauthorized?.();
    }
    return Promise.reject(error);
  },
);
