import {
  keepPreviousData,
  useQuery,
  type UseQueryResult,
} from '@tanstack/react-query';
import { apiClient } from '../../lib/apiClient';
import { useActiveProfile } from '../../context/ActiveProfileContext';
import { jobsFixture } from './fixtures';
import {
  toJobSummary,
  toPagedJobs,
  type PagedJobs,
  type PagedJobsDto,
  type JobFilters,
} from './types';

/**
 * When `VITE_USE_FIXTURES` is set, the hook resolves from the local fixture
 * instead of hitting the dev API. This keeps the page renderable locally before
 * `GET /api/jobs` (P3-JO-3) lands. Defaults off so production and the deployed
 * dev environment always use the real endpoint.
 */
const useFixtures = import.meta.env.VITE_USE_FIXTURES === 'true';

/** Default page size for the jobs list — mirrors the backend default (ADR-007). */
export const DEFAULT_PAGE_SIZE = 25;

/** Pagination request params — 1-based `page`, `pageSize` clamped to 1..100 server-side. */
export interface JobsPagination {
  page: number;
  pageSize: number;
}

async function fetchJobs(
  filters: JobFilters | undefined,
  pagination: JobsPagination,
  includeExpired: boolean,
): Promise<PagedJobs> {
  if (useFixtures) {
    return {
      items: jobsFixture.map(toJobSummary),
      page: pagination.page,
      pageSize: pagination.pageSize,
      total: jobsFixture.length,
      totalPages: jobsFixture.length > 0 ? 1 : 0,
    };
  }

  // Serialize filter params — omit null/undefined/empty values so the backend
  // does not need to distinguish "not sent" from "sent as empty string".
  const params: Record<string, string | string[] | number | boolean> = {
    page: pagination.page,
    page_size: pagination.pageSize,
  };
  // Server defaults to excluding expired jobs; only send the flag when the
  // caller wants expired jobs included (omit otherwise, matching the default).
  if (includeExpired) params['include_expired'] = true;
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

  const { data } = await apiClient.get<PagedJobsDto>('/jobs', { params });
  return toPagedJobs(data);
}

/**
 * Lists jobs for the active profile (ADR-007 offset pagination). The
 * active-profile id, filters, page/pageSize, and `includeExpired` are all
 * part of the cache key, so switching profiles, filters, pages, or the
 * expired-visibility toggle transparently re-scopes the request; the request
 * is disabled until a profile is resolved (the `X-Active-Profile` header is
 * injected by the axios interceptor).
 *
 * Defaults to page 1 / 25 per page when `pagination` is omitted, and to
 * excluding expired jobs when `includeExpired` is omitted — the backend
 * filters expired jobs in SQL so `items`/`total`/`totalPages` stay consistent
 * regardless of the flag (see the pagination fix in job-browser). Uses
 * `placeholderData: keepPreviousData` so the list does not flash empty while
 * switching pages — the previous page's items stay on screen until the new
 * page resolves.
 */
export function useJobs(
  filters?: JobFilters,
  pagination?: JobsPagination,
  includeExpired = false,
): UseQueryResult<PagedJobs> {
  const { activeProfileId } = useActiveProfile();
  const page = pagination?.page ?? 1;
  const pageSize = pagination?.pageSize ?? DEFAULT_PAGE_SIZE;

  return useQuery({
    queryKey: [
      'jobs',
      activeProfileId,
      filters,
      page,
      pageSize,
      includeExpired,
    ],
    queryFn: () => fetchJobs(filters, { page, pageSize }, includeExpired),
    enabled: useFixtures || activeProfileId !== null,
    placeholderData: keepPreviousData,
  });
}
