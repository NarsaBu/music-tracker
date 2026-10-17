import { useState } from 'react';
import { useNavigate } from 'react-router';
import { PRIMARY_TYPE_LABELS } from '@/entities/release/model';
import { syncStateOf, type Subscription } from '@/entities/subscription/model';
import { useSubscriptionsQuery, useUnsubscribeMutation } from '@/entities/subscription/queries';
import { formatDateTime } from '@/shared/lib/format';
import { useDebouncedValue } from '@/shared/lib/useDebouncedValue';

export const PAGE_SIZE = 48;
export const FILTER_DEBOUNCE_MS = 300;

export interface ArtistCardModel {
  mbid: string;
  name: string;
  details: string;
  country: string | null;
  synced: boolean;
  syncText: string;
  typesText: string;
  /** "own" marks a per-artist override, "global" — the value from settings. */
  typesSource: 'own' | 'global';
  featuring: boolean;
  featuringSource: 'own' | 'global';
  unsubscribing: boolean;
}

function toCard(s: Subscription, unsubscribingMbid: string | undefined): ArtistCardModel {
  const { artist, filter } = s;
  return {
    mbid: artist.mbid,
    name: artist.name,
    details: [artist.disambiguation, artist.type].filter(Boolean).join(' · '),
    country: artist.country,
    synced: syncStateOf(s) === 'synced',
    syncText: s.lastSyncedAt
      ? `Проверен ${formatDateTime(s.lastSyncedAt)}`
      : 'Ждёт первой проверки',
    typesText: filter.primaryTypes.value.map((t) => PRIMARY_TYPE_LABELS[t]).join(', ') || '—',
    typesSource: filter.primaryTypes.overridden ? 'own' : 'global',
    featuring: filter.includeFeatured.value,
    featuringSource: filter.includeFeatured.overridden ? 'own' : 'global',
    unsubscribing: unsubscribingMbid === artist.mbid,
  };
}

/** Subscriptions screen ("card catalogue", US-4): filter by name, paging, unsubscribe. */
export function useSubscriptionsController() {
  const navigate = useNavigate();
  const [search, setSearch] = useState('');
  const [page, setPage] = useState(0);
  const debounced = useDebouncedValue(search.trim(), FILTER_DEBOUNCE_MS);

  const list = useSubscriptionsQuery({
    search: debounced,
    limit: PAGE_SIZE,
    offset: page * PAGE_SIZE,
  });
  const unsubscribe = useUnsubscribeMutation();

  const total = list.data?.total ?? 0;
  const pageCount = Math.max(1, Math.ceil(total / PAGE_SIZE));
  const unsubscribingMbid = unsubscribe.isPending ? unsubscribe.variables : undefined;

  return {
    search,
    cards: (list.data?.items ?? []).map((s) => toCard(s, unsubscribingMbid)),
    total,
    isLoading: list.isPending,
    isError: list.isError || unsubscribe.isError,
    /** Nothing subscribed at all (as opposed to nothing matching the filter). */
    isEmpty: list.isSuccess && total === 0 && !debounced,
    page,
    pageCount,

    onSearchChange: (v: string) => {
      setSearch(v);
      setPage(0);
    },
    onPrevPage: () => setPage((p) => Math.max(0, p - 1)),
    onNextPage: () => setPage((p) => Math.min(pageCount - 1, p + 1)),
    onAdd: () => void navigate('/artists/add'),
    onUnsubscribe: (mbid: string) => unsubscribe.mutate(mbid),
  };
}

export type SubscriptionsViewModel = ReturnType<typeof useSubscriptionsController>;
