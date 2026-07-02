import { useQuery, type UseQueryResult } from '@tanstack/react-query';
import { apiClient } from '../../lib/apiClient';
import type { ContactDto } from './types';

/** Lists contacts, optionally filtered by tag, via `GET /api/contacts?tag=`. */
export function useContacts(tag?: string): UseQueryResult<ContactDto[]> {
  return useQuery({
    queryKey: ['contacts', tag ?? null],
    queryFn: async () => {
      const { data } = await apiClient.get<ContactDto[]>('/contacts', {
        params: tag ? { tag } : undefined,
      });
      return data;
    },
  });
}
