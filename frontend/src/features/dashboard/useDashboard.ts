import { useQuery } from '@tanstack/react-query';
import { apiClient } from '../../lib/apiClient';
import { useActiveProfile } from '../../context/ActiveProfileContext';
import type { MatchDto, SkillFrequencyDto, SkillTrendDto, StatsDto } from './types';

/** `GET /api/dashboard/skills/frequency` — ranked skill counts, active-profile scoped. */
export function useSkillFrequency() {
  const { activeProfileId } = useActiveProfile();
  return useQuery({
    queryKey: ['dashboard', 'skills-frequency', activeProfileId],
    queryFn: async () => {
      const { data } = await apiClient.get<SkillFrequencyDto[]>(
        '/dashboard/skills/frequency',
      );
      return data;
    },
    enabled: activeProfileId !== null,
  });
}

/** `GET /api/dashboard/skills/trend` — per-period skill counts, active-profile scoped. */
export function useSkillTrend() {
  const { activeProfileId } = useActiveProfile();
  return useQuery({
    queryKey: ['dashboard', 'skills-trend', activeProfileId],
    queryFn: async () => {
      const { data } = await apiClient.get<SkillTrendDto[]>(
        '/dashboard/skills/trend',
      );
      return data;
    },
    enabled: activeProfileId !== null,
  });
}

/** `GET /api/dashboard/matches` — top match alerts, active-profile scoped. */
export function useDashboardMatches() {
  const { activeProfileId } = useActiveProfile();
  return useQuery({
    queryKey: ['dashboard', 'matches', activeProfileId],
    queryFn: async () => {
      const { data } = await apiClient.get<MatchDto[]>('/dashboard/matches');
      return data;
    },
    enabled: activeProfileId !== null,
  });
}

/** `GET /api/dashboard/stats` — stats cards, active-profile scoped. */
export function useDashboardStats() {
  const { activeProfileId } = useActiveProfile();
  return useQuery({
    queryKey: ['dashboard', 'stats', activeProfileId],
    queryFn: async () => {
      const { data } = await apiClient.get<StatsDto>('/dashboard/stats');
      return data;
    },
    enabled: activeProfileId !== null,
  });
}
