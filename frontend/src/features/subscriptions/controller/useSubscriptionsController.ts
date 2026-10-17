import { useEffect } from 'react';
import { useNavigate, useSearchParams } from 'react-router';
import { artistTypeLabel } from '@/entities/artist/model';
import {
  syncStateOf,
  type Subscription,
  type SubscriptionFilters,
} from '@/entities/subscription/model';
import { useSubscriptionsQuery, useUnsubscribeMutation } from '@/entities/subscription/queries';
import { useI18n, type Messages } from '@/shared/i18n';
import { formatDateTime } from '@/shared/lib/format';
import { useDebouncedValue } from '@/shared/lib/useDebouncedValue';

export const PAGE_SIZES = [24, 48, 96] as const;
export const PAGE_SIZE = 48;
export const FILTER_DEBOUNCE_MS = 300;

/** URL query keys of the catalogue filters, as the API names them. */
export const FILTER_KEYS = ['status', 'type', 'country', 'filter', 'sort'] as const;
export type FilterKey = (typeof FILTER_KEYS)[number];

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

function toCard(
  s: Subscription,
  unsubscribingMbid: string | undefined,
  t: Messages,
): ArtistCardModel {
  const { artist, filter } = s;
  return {
    mbid: artist.mbid,
    name: artist.name,
    details: [artist.disambiguation, artistTypeLabel(artist.type, t.common)]
      .filter(Boolean)
      .join(' · '),
    country: artist.country,
    synced: syncStateOf(s) === 'synced',
    syncText: s.lastSyncedAt
      ? t.common.checkedAt(formatDateTime(s.lastSyncedAt, t.locale))
      : t.common.awaitingFirstCheck,
    typesText:
      filter.primaryTypes.value.map((type) => t.release.primaryTypes[type]).join(', ') || '—',
    typesSource: filter.primaryTypes.overridden ? 'own' : 'global',
    featuring: filter.includeFeatured.value,
    featuringSource: filter.includeFeatured.overridden ? 'own' : 'global',
    unsubscribing: unsubscribingMbid === artist.mbid,
  };
}

function oneOf<T extends string>(value: string | null, allowed: readonly T[]): T | undefined {
  return allowed.find((a) => a === value);
}

/**
 * Subscriptions screen ("card catalogue", US-4): search, filters and paging.
 * The state lives in the URL, so Back, reload and shared links keep the page.
 */
export function useSubscriptionsController() {
  const navigate = useNavigate();
  const { t } = useI18n();
  const [params, setParams] = useSearchParams();

  const search = params.get('q') ?? '';
  const pageSize = PAGE_SIZES.find((s) => String(s) === params.get('size')) ?? PAGE_SIZE;
  const requestedPage = Math.max(0, Math.floor(Number(params.get('page')) || 1) - 1);
  const filters: SubscriptionFilters = {
    status: oneOf(params.get('status'), ['synced', 'pending']),
    type: params.get('type') || undefined,
    country: params.get('country') || undefined,
    filter: oneOf(params.get('filter'), ['own', 'global']),
    sort: oneOf(params.get('sort'), ['name', 'added']),
  };
  const debounced = useDebouncedValue(search.trim(), FILTER_DEBOUNCE_MS);

  const list = useSubscriptionsQuery({
    search: debounced,
    limit: pageSize,
    offset: requestedPage * pageSize,
    ...filters,
  });
  const unsubscribe = useUnsubscribeMutation();

  /** Sets (or, for empty values, removes) query keys; push adds a history entry. */
  const update = (patch: Partial<Record<string, string>>, push = false) =>
    setParams(
      (prev) => {
        const next = new URLSearchParams(prev);
        for (const [key, value] of Object.entries(patch)) {
          if (value) next.set(key, value);
          else next.delete(key);
        }
        return next;
      },
      { replace: !push },
    );

  const total = list.data?.total ?? 0;
  const pageCount = Math.max(1, Math.ceil(total / pageSize));
  // After unsubscribing or a narrower filter the page may be past the end.
  const pastEnd = !!list.data && !list.isPlaceholderData && requestedPage > pageCount - 1;
  useEffect(() => {
    if (!pastEnd) return;
    setParams(
      (prev) => {
        const next = new URLSearchParams(prev);
        if (pageCount > 1) next.set('page', String(pageCount));
        else next.delete('page');
        return next;
      },
      { replace: true },
    );
  }, [pastEnd, pageCount, setParams]);

  const page = Math.min(requestedPage, pageCount - 1);
  const cards = (list.data?.items ?? []).map((s) =>
    toCard(s, unsubscribe.isPending ? unsubscribe.variables : undefined, t),
  );
  const filtered = FILTER_KEYS.some((k) => k !== 'sort' && filters[k]);

  return {
    search,
    filters: {
      status: filters.status ?? '',
      type: filters.type ?? '',
      country: filters.country ?? '',
      filter: filters.filter ?? '',
      sort: filters.sort ?? 'name',
    } satisfies Record<FilterKey, string>,
    facets: list.data?.facets ?? { countries: [], types: [] },
    cards,
    total,
    isLoading: list.isPending,
    isError: list.isError || unsubscribe.isError,
    /** Nothing subscribed at all (as opposed to nothing matching the search or filters). */
    isEmpty: list.isSuccess && total === 0 && !debounced && !filtered,
    /** Some filter (not the order) narrows the list. */
    filtered,
    page,
    pageCount,
    pageSize,
    /** 1-based positions of the shown cards, for "49–96 of 716". */
    range: { from: total ? page * pageSize + 1 : 0, to: page * pageSize + cards.length },

    onSearchChange: (v: string) => update({ q: v, page: undefined }),
    onFilterChange: (key: FilterKey, value: string) =>
      update(
        { [key]: key === 'sort' && value === 'name' ? undefined : value, page: undefined },
        true,
      ),
    onResetFilters: () =>
      update(
        {
          q: undefined,
          page: undefined,
          ...Object.fromEntries(FILTER_KEYS.map((k) => [k, undefined])),
        },
        true,
      ),
    onPageChange: (p: number) => update({ page: p > 0 ? String(p + 1) : undefined }, true),
    onPageSizeChange: (size: number) =>
      update({ size: size === PAGE_SIZE ? undefined : String(size), page: undefined }, true),
    onAdd: () => void navigate('/artists/add'),
    onUnsubscribe: (mbid: string) => unsubscribe.mutate(mbid),
  };
}

export type SubscriptionsViewModel = ReturnType<typeof useSubscriptionsController>;
