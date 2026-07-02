import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { apiClient } from '../../lib/apiClient';
import {
  isTerminalStatus,
  type ScrapeRunDetailDto,
  type ScrapeRunListResponseDto,
} from './types';

/**
 * Lists recent pipeline runs across all profiles via `GET /api/pipeline/runs`
 * (not gated to the active profile server-side). Used for run history; has
 * no per-board breakdown, so it is not enough for live progress display.
 */
export function usePipelineRunList() {
  return useQuery({
    queryKey: ['pipeline-runs'],
    queryFn: async () => {
      const { data } = await apiClient.get<ScrapeRunListResponseDto>(
        '/pipeline/runs',
      );
      return data.runs;
    },
  });
}

/**
 * Polls a single run's detail (including per-board progress) via
 * `GET /api/pipeline/runs/{id}`. Polling stops automatically once the run
 * reaches a terminal status.
 */
export function usePipelineRunDetail(runId: string | null) {
  return useQuery({
    queryKey: ['pipeline-run', runId],
    queryFn: async () => {
      const { data } = await apiClient.get<ScrapeRunDetailDto>(
        `/pipeline/runs/${runId}`,
      );
      return data;
    },
    enabled: runId !== null,
    refetchInterval: (query) => {
      const data = query.state.data as ScrapeRunDetailDto | undefined;
      if (data && isTerminalStatus(data.status)) return false;
      return 2000;
    },
  });
}

/** Triggers an on-demand run via `POST /api/pipeline/runs` for the active profile. */
export function useTriggerPipelineRunMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async () => {
      const { data } = await apiClient.post<{ run_id: string }>(
        '/pipeline/runs',
      );
      return data.run_id;
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['pipeline-runs'] });
    },
  });
}
