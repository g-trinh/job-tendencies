import { useQuery, type UseQueryResult } from '@tanstack/react-query';
import { apiClient } from '../../lib/apiClient';
import { useActiveProfile } from '../../context/ActiveProfileContext';
import { jobDetailFixture } from './fixtures';
import { toJobDetail, type JobDetail, type JobDetailDto } from './types';

const useFixtures = import.meta.env.VITE_USE_FIXTURES === 'true';

async function fetchJobDetail(id: string): Promise<JobDetail> {
  if (useFixtures) {
    return toJobDetail(jobDetailFixture);
  }
  const { data } = await apiClient.get<JobDetailDto>(`/jobs/${id}`);
  return toJobDetail(data);
}

/**
 * Fetches full detail for a single job. The active-profile id is part of the
 * cache key because job visibility (and application status) is profile-scoped.
 */
export function useJobDetail(id: string): UseQueryResult<JobDetail> {
  const { activeProfileId } = useActiveProfile();

  return useQuery({
    queryKey: ['job', id, activeProfileId],
    queryFn: () => fetchJobDetail(id),
    enabled: useFixtures || activeProfileId !== null,
  });
}
