import { useMutation, useQueryClient } from '@tanstack/react-query';
import { apiClient } from '../../lib/apiClient';
import { useActiveProfile } from '../../context/ActiveProfileContext';
import type { ApplicationStatus } from './types';

interface ApplicationResponseDto {
  status: ApplicationStatus;
  updated_at: string;
}

/**
 * Mutation hook for updating the application status of a job via
 * `PATCH /api/jobs/{id}/application`. On success, invalidates the jobs list
 * and the job detail cache so they re-fetch with the updated status.
 *
 * Usage:
 *   const { mutate, isPending } = useApplicationMutation(jobId);
 *   mutate('applied');
 */
export function useApplicationMutation(jobId: string) {
  const queryClient = useQueryClient();
  const { activeProfileId } = useActiveProfile();

  return useMutation({
    mutationFn: async (status: ApplicationStatus): Promise<ApplicationResponseDto> => {
      const { data } = await apiClient.patch<ApplicationResponseDto>(
        `/jobs/${jobId}/application`,
        { status },
      );
      return data;
    },
    onSuccess: () => {
      // Invalidate scoped caches so the list + detail both re-fetch fresh status.
      void queryClient.invalidateQueries({ queryKey: ['jobs', activeProfileId] });
      void queryClient.invalidateQueries({ queryKey: ['job', jobId, activeProfileId] });
    },
  });
}
