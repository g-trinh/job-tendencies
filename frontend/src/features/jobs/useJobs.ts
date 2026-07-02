import { useQuery, type UseQueryResult } from '@tanstack/react-query';
import { apiClient } from '../../lib/apiClient';
import { useActiveProfile } from '../../context/ActiveProfileContext';
import { jobsFixture } from './fixtures';
import {
  toJobSummary,
  type JobSummary,
  type JobSummaryDto,
  type JobFilters,
} from './types';

/**
 * When `VITE_USE_FIXTURES` is set, the hook resolves from the local fixture
 * instead of hitting the dev API. This keeps the page renderable locally before
 * `GET /api/jobs` (P3-JO-3) lands. Defaults off so production and the deployed
 * dev environment always use the real endpoint.
 */
const useFixtures = import.meta.env.VITE_USE_FIXTURES === 'true';

async function fetchJobs(filters?: JobFilters): Promise<JobSummary[]> {
  if (useFixtures) {
    return jobsFixture.map(toJobSummary);
  }

  // Serialize filter params — omit null/undefined/empty values so the backend
  // does not need to distinguish "not sent" from "sent as empty string".
  const params: Record<string, string | string[] | number> = {};
  if (filters) {
    if (filters.skills?.length) params['skills'] = filters.skills;
    if (filters.remote_policy) params['remote_policy'] = filters.remote_policy;
    if (filters.contract_type) params['contract_type'] = filters.contract_type;
    if (filters.salary_min != null) params['salary_min'] = filters.salary_min;
    if (filters.salary_max != null) params['salary_max'] = filters.salary_max;
    if (filters.location) params['location'] = filters.location;
    if (filters.board_id) params['board_id'] = filters.board_id;
    if (filters.since) params['since'] = filters.since;
    if (filters.confidence_min != null)
      params['confidence_min'] = filters.confidence_min;
    if (filters.sort) params['sort'] = filters.sort;
    if (filters.sort_dir) params['sort_dir'] = filters.sort_dir;
  }

  const { data } = await apiClient.get<JobSummaryDto[]>('/jobs', { params });
  return data.map(toJobSummary);
}

/**
 * Lists jobs for the active profile. The active-profile id is part of the cache
 * key, so switching profiles transparently re-scopes the list; the request is
 * disabled until a profile is resolved (the `X-Active-Profile` header is injected
 * by the axios interceptor). Pass `filters` to narrow or sort the result set.
 */
export function useJobs(filters?: JobFilters): UseQueryResult<JobSummary[]> {
  const { activeProfileId } = useActiveProfile();

  return useQuery({
    queryKey: ['jobs', activeProfileId, filters],
    queryFn: () => fetchJobs(filters),
    enabled: useFixtures || activeProfileId !== null,
  });
}
