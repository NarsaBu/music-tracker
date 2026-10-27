import { keepPreviousData, useMutation, useQuery } from '@tanstack/react-query';
import { httpClient } from '@/shared/api';
import { createReleaseApi } from './api';
import type { CreditRole } from './model';

const releaseApi = createReleaseApi(httpClient);

export const releaseKeys = {
  all: ['release-groups'] as const,
  byArtist: (mbid: string) => [...releaseKeys.all, 'artist', mbid] as const,
  byArtistRole: (mbid: string, role: CreditRole | null) =>
    [...releaseKeys.byArtist(mbid), role ?? 'all'] as const,
  one: (mbid: string) => [...releaseKeys.all, 'one', mbid] as const,
  reviews: (mbid: string) => [...releaseKeys.all, 'reviews', mbid] as const,
  info: (mbid: string) => [...releaseKeys.all, 'info', mbid] as const,
};

export function useArtistReleaseGroupsQuery(mbid: string, role: CreditRole | null) {
  return useQuery({
    queryKey: releaseKeys.byArtistRole(mbid, role),
    queryFn: () => releaseApi.listByArtist(mbid, role),
    placeholderData: keepPreviousData,
  });
}

export function useReleaseQuery(mbid: string) {
  return useQuery({ queryKey: releaseKeys.one(mbid), queryFn: () => releaseApi.get(mbid) });
}

/** Genres, tracklist and credited artists from MusicBrainz (cached on the server for a week). */
export function useReleaseInfoQuery(mbid: string) {
  return useQuery({
    queryKey: releaseKeys.info(mbid),
    queryFn: () => releaseApi.info(mbid),
    staleTime: 60 * 60_000,
    retry: false,
  });
}

/** Looks for a cover big enough to download; on demand, it costs archive requests. */
export function useCoverInfoMutation() {
  return useMutation({ mutationFn: (mbid: string) => releaseApi.coverInfo(mbid) });
}

/** CritiqueBrainz reviews; slow upstream, cached on the server for a day. */
export function useReviewsQuery(mbid: string) {
  return useQuery({
    queryKey: releaseKeys.reviews(mbid),
    queryFn: () => releaseApi.reviews(mbid),
    staleTime: 60 * 60_000,
    retry: false,
  });
}
