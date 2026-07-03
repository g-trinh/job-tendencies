import { useQuery, type UseQueryResult } from '@tanstack/react-query';
import { apiClient } from '../../lib/apiClient';
import type { BoardDto } from './types';

/** Lists every board (not profile-scoped — boards are global). */
export function useBoards(): UseQueryResult<BoardDto[]> {
  return useQuery({
    queryKey: ['boards'],
    queryFn: async () => {
      const { data } = await apiClient.get<BoardDto[]>('/boards');
      return data;
    },
  });
}
