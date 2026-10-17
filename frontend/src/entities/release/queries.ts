import { keepPreviousData, useQuery } from '@tanstack/react-query';
import { httpClient } from '@/shared/api';
import { createReleaseApi } from './api';
import type { CreditRole } from './model';

const releaseApi = createReleaseApi(httpClient);

export const releaseKeys = {
  all: ['release-groups'] as const,
  byArtist: (mbid: string) => [...releaseKeys.all, 'artist', mbid] as const,
  byArtistRole: (mbid: string, role: CreditRole | null) =>
    [...releaseKeys.byArtist(mbid), role ?? 'all'] as const,
};

export function useArtistReleaseGroupsQuery(mbid: string, role: CreditRole | null) {
  return useQuery({
    queryKey: releaseKeys.byArtistRole(mbid, role),
    queryFn: () => releaseApi.listByArtist(mbid, role),
    placeholderData: keepPreviousData,
  });
}
