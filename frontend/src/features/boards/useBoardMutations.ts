import { useMutation, useQueryClient } from '@tanstack/react-query';
import { apiClient } from '../../lib/apiClient';
import type { AdapterDto, BoardDto } from './types';

function useInvalidateBoards() {
  const queryClient = useQueryClient();
  return () => void queryClient.invalidateQueries({ queryKey: ['boards'] });
}

/** Creates a board (enabled by default) via `POST /api/boards`. */
export function useCreateBoardMutation() {
  const invalidate = useInvalidateBoards();
  return useMutation({
    mutationFn: async (body: { name: string; base_url: string }) => {
      const { data } = await apiClient.post<BoardDto>('/boards', body);
      return data;
    },
    onSuccess: invalidate,
  });
}

/** Updates a board (name/base_url/enabled) via `PUT /api/boards/{id}`. */
export function useUpdateBoardMutation() {
  const invalidate = useInvalidateBoards();
  return useMutation({
    mutationFn: async (body: {
      id: string;
      name: string;
      base_url: string;
      enabled: boolean;
    }) => {
      const { data } = await apiClient.put<BoardDto>(`/boards/${body.id}`, {
        name: body.name,
        base_url: body.base_url,
        enabled: body.enabled,
      });
      return data;
    },
    onSuccess: invalidate,
  });
}

/** Deletes a board via `DELETE /api/boards/{id}`. */
export function useDeleteBoardMutation() {
  const invalidate = useInvalidateBoards();
  return useMutation({
    mutationFn: async (id: string) => {
      await apiClient.delete(`/boards/${id}`);
    },
    onSuccess: invalidate,
  });
}

/** Generates an adapter draft via `POST /api/boards/{id}/adapter/generate`. */
export function useGenerateAdapterMutation(boardId: string) {
  const invalidate = useInvalidateBoards();
  return useMutation({
    mutationFn: async (exampleResponse: string) => {
      const { data } = await apiClient.post<AdapterDto>(
        `/boards/${boardId}/adapter/generate`,
        { example_response: exampleResponse },
      );
      return data;
    },
    onSuccess: invalidate,
  });
}

/** Approves the latest draft adapter via `POST /api/boards/{id}/adapter/approve`. */
export function useApproveAdapterMutation(boardId: string) {
  const invalidate = useInvalidateBoards();
  return useMutation({
    mutationFn: async () => {
      const { data } = await apiClient.post<AdapterDto>(
        `/boards/${boardId}/adapter/approve`,
      );
      return data;
    },
    onSuccess: invalidate,
  });
}
