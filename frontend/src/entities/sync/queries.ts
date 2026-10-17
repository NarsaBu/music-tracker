import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { httpClient } from '@/shared/api';
import { createSyncApi } from './api';
import { isRunning } from './model';

const syncApi = createSyncApi(httpClient);

export const syncKeys = {
  latest: ['sync', 'latest'] as const,
};

export const SYNC_POLL_MS = 2_000;

/** Latest run; polls while it is in progress to drive the VU meter. */
export function useLatestSyncRunQuery() {
  return useQuery({
    queryKey: syncKeys.latest,
    queryFn: syncApi.latest,
    refetchInterval: (q) => (isRunning(q.state.data) ? SYNC_POLL_MS : false),
  });
}

export function useStartSyncMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: syncApi.start,
    onSuccess: (run) => queryClient.setQueryData(syncKeys.latest, run),
  });
}
