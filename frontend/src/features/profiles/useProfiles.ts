import { useQuery, type UseQueryResult } from '@tanstack/react-query';
import { apiClient } from '../../lib/apiClient';
import { toProfile, type Profile, type ProfileDto } from './types';

/** Lists every profile (not scoped to the active profile — this is the switcher's source). */
export function useProfiles(): UseQueryResult<Profile[]> {
  return useQuery({
    queryKey: ['profiles'],
    queryFn: async () => {
      const { data } = await apiClient.get<ProfileDto[]>('/profiles');
      return data.map(toProfile);
    },
  });
}
