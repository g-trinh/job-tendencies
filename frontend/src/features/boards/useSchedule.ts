import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { apiClient } from '../../lib/apiClient';
import type { ScheduleDto } from './types';

/** Reads the single global cron schedule via `GET /api/schedule`. */
export function useSchedule() {
  return useQuery({
    queryKey: ['schedule'],
    queryFn: async () => {
      const { data } = await apiClient.get<ScheduleDto>('/schedule');
      return data;
    },
  });
}

/** Updates the global cron schedule via `PUT /api/schedule`. */
export function useUpdateScheduleMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (cron: string) => {
      const { data } = await apiClient.put<ScheduleDto>('/schedule', { cron });
      return data;
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['schedule'] });
    },
  });
}
