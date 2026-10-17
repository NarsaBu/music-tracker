import { useQuery } from '@tanstack/react-query';
import { httpClient } from '@/shared/api';
import { createSystemApi } from './api';

const systemApi = createSystemApi(httpClient);

export const systemKeys = {
  health: ['system', 'health'] as const,
};

export type BackendStatus = 'checking' | 'online' | 'offline';

/** Polls backend liveness; drives the ONLINE/OFFLINE LCD in the rack header. */
export function useBackendStatus(): BackendStatus {
  const { isPending, isError } = useQuery({
    queryKey: systemKeys.health,
    queryFn: systemApi.health,
    refetchInterval: 30_000,
    retry: false,
  });
  if (isPending) return 'checking';
  return isError ? 'offline' : 'online';
}
