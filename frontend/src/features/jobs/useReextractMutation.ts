import { useMutation, type UseMutationResult } from '@tanstack/react-query';
import { apiClient } from '../../lib/apiClient';

interface ReextractResponseDto {
  status: string;
}

/**
 * Mutation hook for `POST /api/jobs/{id}/reextract` (P5-4): re-publishes
 * `listing.extract` for the job's retained raw listing(s) so an improved
 * extractor can reprocess it. The backend returns 202 Accepted immediately —
 * re-extraction itself runs asynchronously in extract-worker, so this hook
 * does not invalidate the job query cache; the job's fields only change once
 * the worker finishes and the user re-fetches later.
 *
 * Usage:
 *   const { mutate, isPending, isSuccess, isError } = useReextractMutation(jobId);
 *   mutate();
 */
export function useReextractMutation(
  jobId: string,
): UseMutationResult<ReextractResponseDto, unknown, void> {
  return useMutation({
    mutationFn: async (): Promise<ReextractResponseDto> => {
      const { data } = await apiClient.post<ReextractResponseDto>(
        `/jobs/${jobId}/reextract`,
      );
      return data;
    },
  });
}
