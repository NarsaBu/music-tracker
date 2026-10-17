import { keepPreviousData, useQuery } from '@tanstack/react-query';
import { httpClient } from '@/shared/api';
import { createArtistApi } from './api';

const artistApi = createArtistApi(httpClient);

export const artistKeys = {
  all: ['artists'] as const,
  search: (query: string) => [...artistKeys.all, 'search', query] as const,
};

/** MusicBrainz search through the backend. Disabled for a blank query. */
export function useArtistSearchQuery(query: string) {
  const q = query.trim();
  return useQuery({
    queryKey: artistKeys.search(q),
    queryFn: () => artistApi.search(q),
    enabled: q.length > 0,
    // Keep the previous "stations" on screen while the next query is tuning in.
    placeholderData: keepPreviousData,
    // MusicBrainz allows 1 rps; a failed search is retried by the user, not automatically.
    retry: false,
  });
}
