import { useMutation, useQueryClient } from '@tanstack/react-query';
import { apiClient } from '../../lib/apiClient';
import { useActiveProfile } from '../../context/ActiveProfileContext';
import type { ApplicationStatus, PagedJobs } from './types';

interface ApplicationResponseDto {
  status: ApplicationStatus;
  updated_at: string;
}

/**
 * Mutation hook for updating the application status of a job via
 * `PATCH /api/jobs/{id}/application`. Applies an optimistic update to the
 * cached jobs list (so a kanban card moves columns immediately) and rolls
 * back on error; on settle, invalidates the jobs list and job detail cache
 * so both re-fetch with the server-confirmed status.
 *
 * Usage:
 *   const { mutate, isPending } = useApplicationMutation(jobId);
 *   mutate('applied');
 */
export function useApplicationMutation(jobId: string) {
  const queryClient = useQueryClient();
  const { activeProfileId } = useActiveProfile();

  return useMutation({
    mutationFn: async (
      status: ApplicationStatus,
    ): Promise<ApplicationResponseDto> => {
      const { data } = await apiClient.patch<ApplicationResponseDto>(
        `/jobs/${jobId}/application`,
        { status },
      );
      return data;
    },
    onMutate: async (status: ApplicationStatus) => {
      await queryClient.cancelQueries({ queryKey: ['jobs', activeProfileId] });

      const previousLists = queryClient.getQueriesData<PagedJobs>({
        queryKey: ['jobs', activeProfileId],
      });

      previousLists.forEach(([key, paged]) => {
        if (!paged) return;
        queryClient.setQueryData<PagedJobs>(key, {
          ...paged,
          items: paged.items.map((job) =>
            job.id === jobId ? { ...job, applicationStatus: status } : job,
          ),
        });
      });

      return { previousLists };
    },
    onError: (_err, _status, context) => {
      // Roll back every list snapshot captured in onMutate.
      context?.previousLists.forEach(([key, jobs]) => {
        queryClient.setQueryData(key, jobs);
      });
    },
    onSettled: () => {
      // Re-fetch to reconcile with server-confirmed state.
      void queryClient.invalidateQueries({
        queryKey: ['jobs', activeProfileId],
      });
      void queryClient.invalidateQueries({
        queryKey: ['job', jobId, activeProfileId],
      });
    },
  });
}
