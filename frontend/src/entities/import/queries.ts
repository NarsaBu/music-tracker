import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { subscriptionKeys } from '@/entities/subscription/queries';
import { httpClient } from '@/shared/api';
import { createImportApi } from './api';
import type { ImportItemStatus, ResolveRequest } from './model';

const importApi = createImportApi(httpClient);

export const importKeys = {
  all: ['imports'] as const,
  list: ['imports', 'list'] as const,
  one: (id: string) => ['imports', 'one', id] as const,
  items: (id: string, statuses: readonly ImportItemStatus[]) =>
    ['imports', 'items', id, statuses.join(',')] as const,
};

export const IMPORT_POLL_MS = 2_000;

/** Recent imports; polls while any of them is still being processed. */
export function useImportsQuery() {
  return useQuery({
    queryKey: importKeys.list,
    queryFn: importApi.list,
    refetchInterval: (q) => (q.state.data?.some((j) => !j.finished) ? IMPORT_POLL_MS : false),
  });
}

/** One import; polls until the worker is done with it. */
export function useImportQuery(id: string) {
  return useQuery({
    queryKey: importKeys.one(id),
    queryFn: () => importApi.get(id),
    refetchInterval: (q) => (q.state.data && !q.state.data.finished ? IMPORT_POLL_MS : false),
  });
}

export interface ImportItemsQueryOptions {
  /** Poll interval in ms while lines are still being matched; false = no polling. */
  pollMs?: number | false;
}

/** The job's lines with any of the statuses; none = all lines. */
export function useImportItemsQuery(
  id: string,
  statuses: readonly ImportItemStatus[],
  { pollMs = false }: ImportItemsQueryOptions = {},
) {
  return useQuery({
    queryKey: importKeys.items(id, statuses),
    queryFn: () => importApi.items(id, statuses),
    refetchInterval: pollMs,
  });
}

export function useUploadImportMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (file: File) => importApi.upload(file),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: importKeys.list }),
  });
}

export function useResolveItemMutation(id: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ itemId, req }: { itemId: string; req: ResolveRequest }) =>
      importApi.resolve(id, itemId, req),
    onSuccess: () =>
      Promise.all([
        queryClient.invalidateQueries({ queryKey: importKeys.all }),
        queryClient.invalidateQueries({ queryKey: subscriptionKeys.all }),
      ]),
  });
}
