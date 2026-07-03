import { useMutation, useQueryClient } from '@tanstack/react-query';
import { apiClient } from '../../lib/apiClient';
import type { ContactDto, ContactWriteRequest } from './types';

function useInvalidateContacts() {
  const queryClient = useQueryClient();
  return () => void queryClient.invalidateQueries({ queryKey: ['contacts'] });
}

/**
 * Adds a contact via `POST /api/contacts`. This is an UPSERT keyed on
 * dedup_key (email or linkedin_url) server-side: submitting a contact whose
 * email/LinkedIn URL already exists merges into the existing record (200)
 * instead of creating a duplicate (201) — never errors on a "duplicate".
 */
export function useUpsertContactMutation() {
  const invalidate = useInvalidateContacts();
  return useMutation({
    mutationFn: async (body: ContactWriteRequest) => {
      const { data, status } = await apiClient.post<ContactDto>(
        '/contacts',
        body,
      );
      return { contact: data, created: status === 201 };
    },
    onSuccess: invalidate,
  });
}

/** Updates a contact (tags, notes, and other editable fields) via `PUT /api/contacts/{id}`. */
export function useUpdateContactMutation() {
  const invalidate = useInvalidateContacts();
  return useMutation({
    mutationFn: async (body: ContactWriteRequest & { id: string }) => {
      const { id, ...rest } = body;
      const { data } = await apiClient.put<ContactDto>(
        `/contacts/${id}`,
        rest,
      );
      return data;
    },
    onSuccess: invalidate,
  });
}

/** Deletes a contact via `DELETE /api/contacts/{id}`. */
export function useDeleteContactMutation() {
  const invalidate = useInvalidateContacts();
  return useMutation({
    mutationFn: async (id: string) => {
      await apiClient.delete(`/contacts/${id}`);
    },
    onSuccess: invalidate,
  });
}
