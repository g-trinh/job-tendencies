import { useMutation, useQueryClient } from '@tanstack/react-query';
import { apiClient } from '../../lib/apiClient';
import {
  toProfile,
  type Profile,
  type ProfileConditionsDto,
  type ProfileDto,
  type ProfileWeightsDto,
} from './types';

/** Invalidates every profile-related cache after a mutation settles. */
function useInvalidateProfiles() {
  const queryClient = useQueryClient();
  return () => {
    void queryClient.invalidateQueries({ queryKey: ['profiles'] });
    void queryClient.invalidateQueries({ queryKey: ['active-profile'] });
  };
}

/** Creates a new (inactive) profile via `POST /api/profiles`. */
export function useCreateProfileMutation() {
  const invalidate = useInvalidateProfiles();
  return useMutation({
    mutationFn: async (body: {
      name: string;
      location: string;
      search_keywords: string[];
    }): Promise<Profile> => {
      const { data } = await apiClient.post<ProfileDto>('/profiles', body);
      return toProfile(data);
    },
    onSuccess: invalidate,
  });
}

/** Updates skills + seniority via `PATCH /api/profiles/{id}/identity`. */
export function useUpdateIdentityMutation(profileId: string) {
  const invalidate = useInvalidateProfiles();
  return useMutation({
    mutationFn: async (body: {
      skills: string[];
      seniority: string;
    }): Promise<Profile> => {
      const { data } = await apiClient.patch<ProfileDto>(
        `/profiles/${profileId}/identity`,
        body,
      );
      return toProfile(data);
    },
    onSuccess: invalidate,
  });
}

/** Imports a LinkedIn PDF export via `POST /api/profiles/{id}/identity/import`. */
export function useImportIdentityMutation(profileId: string) {
  const invalidate = useInvalidateProfiles();
  return useMutation({
    mutationFn: async (file: File): Promise<Profile> => {
      const form = new FormData();
      form.append('file', file);
      const { data } = await apiClient.post<ProfileDto>(
        `/profiles/${profileId}/identity/import`,
        form,
        { headers: { 'Content-Type': 'multipart/form-data' } },
      );
      return toProfile(data);
    },
    onSuccess: invalidate,
  });
}

/** Updates deal-breaker/preference conditions via `PUT /api/profiles/{id}/conditions`. */
export function useUpdateConditionsMutation(profileId: string) {
  const invalidate = useInvalidateProfiles();
  return useMutation({
    mutationFn: async (body: ProfileConditionsDto): Promise<Profile> => {
      const { data } = await apiClient.put<ProfileDto>(
        `/profiles/${profileId}/conditions`,
        body,
      );
      return toProfile(data);
    },
    onSuccess: invalidate,
  });
}

/** Updates fit-score weights via `PUT /api/profiles/{id}/weights`. */
export function useUpdateWeightsMutation(profileId: string) {
  const invalidate = useInvalidateProfiles();
  return useMutation({
    mutationFn: async (body: ProfileWeightsDto): Promise<Profile> => {
      const { data } = await apiClient.put<ProfileDto>(
        `/profiles/${profileId}/weights`,
        body,
      );
      return toProfile(data);
    },
    onSuccess: invalidate,
  });
}
