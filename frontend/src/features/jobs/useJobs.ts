import { useQuery, type UseQueryResult } from '@tanstack/react-query';
import { apiClient } from '../../lib/apiClient';
import { useActiveProfile } from '../../context/ActiveProfileContext';
import { jobsFixture } from './fixtures';
import { toJobSummary, type JobSummary, type JobSummaryDto } from './types';

/**
 * When `VITE_USE_FIXTURES` is set, the hook resolves from the local fixture
 * instead of hitting the dev API. This keeps the walking-skeleton page renderable
 * locally before `GET /api/jobs` (P2-BE-5) lands. Defaults off so production and
 * the deployed dev environment always use the real endpoint.
 */
const useFixtures = import.meta.env.VITE_USE_FIXTURES === 'true';

async function fetchJobs(): Promise<JobSummary[]> {
  if (useFixtures) {
    return jobsFixture.map(toJobSummary);
  }
  const { data } = await apiClient.get<JobSummaryDto[]>('/jobs');
  return data.map(toJobSummary);
}

/**
 * Lists jobs for the active profile. The active-profile id is part of the cache
 * key, so switching profiles transparently re-scopes the list; the request is
 * disabled until a profile is resolved (the `X-Active-Profile` header is injected
 * by the axios interceptor).
 */
export function useJobs(): UseQueryResult<JobSummary[]> {
  const { activeProfileId } = useActiveProfile();

  return useQuery({
    queryKey: ['jobs', activeProfileId],
    queryFn: fetchJobs,
    enabled: useFixtures || activeProfileId !== null,
  });
}
