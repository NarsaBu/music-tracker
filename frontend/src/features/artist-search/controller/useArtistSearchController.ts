import { useState } from 'react';
import { artistTypeLabel, formatLifeSpan } from '@/entities/artist/model';
import { useArtistSearchQuery } from '@/entities/artist/queries';
import { useSubscribeMutation } from '@/entities/subscription/queries';
import { ApiError } from '@/shared/api';
import { useI18n, type Messages } from '@/shared/i18n';
import { useDebouncedValue } from '@/shared/lib/useDebouncedValue';

export const SEARCH_DEBOUNCE_MS = 400;

export type TunerStatus = 'idle' | 'tuning' | 'error' | 'empty' | 'results';

export interface StationRow {
  mbid: string;
  name: string;
  details: string;
  country: string | null;
  years: string;
  score: number;
  subscribed: boolean;
  subscribing: boolean;
}

function errorText(error: unknown, t: Messages): string {
  if (error instanceof ApiError && error.status === 503) return t.common.musicbrainzDown;
  return t.search.requestFailed;
}

/** Artist search screen ("tuner", US-1): debounced search and one-click subscribe. */
export function useArtistSearchController() {
  const { t } = useI18n();
  const [query, setQuery] = useState('');
  const [includeFeatured, setIncludeFeatured] = useState(false);
  const debounced = useDebouncedValue(query.trim(), SEARCH_DEBOUNCE_MS);

  const search = useArtistSearchQuery(debounced);
  const subscribe = useSubscribeMutation();

  // Typing ahead of the debounce counts as tuning: the shown results are stale.
  const typing = query.trim() !== debounced;
  const status: TunerStatus = !query.trim()
    ? 'idle'
    : typing || search.isFetching
      ? 'tuning'
      : search.isError
        ? 'error'
        : search.data?.length
          ? 'results'
          : 'empty';

  const stations: StationRow[] = query.trim()
    ? (search.data ?? []).map((a) => ({
        mbid: a.mbid,
        name: a.name,
        details: [a.disambiguation, artistTypeLabel(a.type, t.common)].filter(Boolean).join(' · '),
        country: a.country,
        years: formatLifeSpan(a.lifeSpan),
        score: a.score,
        subscribed: a.subscribed,
        subscribing: subscribe.isPending && subscribe.variables?.mbid === a.mbid,
      }))
    : [];

  const lcdText = {
    idle: t.common.lcd.noSignal,
    tuning: t.common.lcd.tuning,
    error: t.common.lcd.error,
    empty: t.search.stations(0),
    results: t.search.stations(stations.length),
  }[status];

  return {
    query,
    status,
    lcdText,
    stations,
    includeFeatured,
    errorText: search.isError
      ? errorText(search.error, t)
      : subscribe.isError
        ? errorText(subscribe.error, t)
        : undefined,

    onQueryChange: setQuery,
    onIncludeFeaturedChange: setIncludeFeatured,
    onRetry: () => void search.refetch(),
    onSubscribe: (mbid: string) =>
      // Off = no override: the global "featuring" setting applies.
      subscribe.mutate({ mbid, includeFeatured: includeFeatured || undefined }),
  };
}

export type ArtistSearchViewModel = ReturnType<typeof useArtistSearchController>;
