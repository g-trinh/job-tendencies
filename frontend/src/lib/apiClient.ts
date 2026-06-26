import axios from 'axios';

/**
 * Module-level active-profile id. Updated by ActiveProfileProvider via
 * setActiveProfileId() below. Kept outside React so it is readable by the
 * axios interceptor without a hook and is trivially testable in isolation.
 */
let _activeProfileId: string | null = null;

/** Called by ActiveProfileProvider whenever the active profile changes. */
export function setActiveProfileId(id: string | null): void {
  _activeProfileId = id;
}

export const apiClient = axios.create({
  baseURL: '/api',
});

apiClient.interceptors.request.use((config) => {
  if (_activeProfileId !== null) {
    config.headers['X-Active-Profile'] = _activeProfileId;
  }
  return config;
});
