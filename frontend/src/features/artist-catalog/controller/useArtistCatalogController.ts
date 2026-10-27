import { useQueryClient } from '@tanstack/react-query';
import { useEffect, useRef, useState } from 'react';
import { artistTypeLabel } from '@/entities/artist/model';
import {
  CREDIT_ROLES,
  formatPartialDate,
  formatRating,
  PRIMARY_TYPES,
  type CreditRole,
  type PrimaryType,
} from '@/entities/release/model';
import { releaseKeys, useArtistReleaseGroupsQuery } from '@/entities/release/queries';
import { useRequestSyncMutation, useSubscriptionQuery } from '@/entities/subscription/queries';
import { ApiError } from '@/shared/api';
import { useI18n } from '@/shared/i18n';
import { formatDateTime } from '@/shared/lib/format';

export type RoleFilter = CreditRole | 'all';

export const ROLE_FILTERS: readonly RoleFilter[] = ['all', ...CREDIT_ROLES];

export type TypeFilter = PrimaryType | 'all';

export const TYPE_FILTERS: readonly TypeFilter[] = ['all', ...PRIMARY_TYPES];

export const CHECK_POLL_MS = 2_000;
/** A requested check that has not landed by then is reported as stuck (the job keeps retrying). */
export const CHECK_TIMEOUT_MS = 3 * 60_000;

export interface RecordModel {
  mbid: string;
  title: string;
  credit: string;
  date: string;
  coverUrl: string;
  /** The release page in the app: ratings and reviews. */
  releasePath: string;
  /** "★ 4,6 · 89"; null when unrated on MusicBrainz. */
  ratingText: string | null;
  typeLabel: string | null;
  secondaryTypes: string[];
  featured: boolean;
  announcement: boolean;
  /** Discovered after the baseline: this is what notifications are about. */
  fresh: boolean;
}

/**
 * Artist page (US-10): stored catalog, role filter (on the server) and type filter
 * (here: the catalog comes whole), "check now".
 */
export function useArtistCatalogController(mbid: string) {
  const queryClient = useQueryClient();
  const { t } = useI18n();
  const [role, setRole] = useState<RoleFilter>('all');
  const [type, setType] = useState<TypeFilter>('all');
  // lastSyncedAt seen when a check was requested; undefined = no check in flight.
  const [checkFrom, setCheckFrom] = useState<string | null | undefined>(undefined);
  const [checkStuck, setCheckStuck] = useState(false);

  const subscription = useSubscriptionQuery(mbid, {
    pollMs: checkFrom !== undefined && !checkStuck ? CHECK_POLL_MS : false,
  });
  const releases = useArtistReleaseGroupsQuery(mbid, role === 'all' ? null : role);
  const requestSync = useRequestSyncMutation();

  const lastSyncedAt = subscription.data?.lastSyncedAt ?? null;
  const checking =
    requestSync.isPending || (checkFrom !== undefined && lastSyncedAt === checkFrom && !checkStuck);

  // A new sync time means a new catalog. undefined = still loading, which is not a change;
  // null → date is: the first sync just filled the catalog.
  const loadedSyncedAt = subscription.data === undefined ? undefined : lastSyncedAt;
  const prevSynced = useRef(loadedSyncedAt);
  useEffect(() => {
    if (prevSynced.current !== undefined && prevSynced.current !== loadedSyncedAt) {
      void queryClient.invalidateQueries({ queryKey: releaseKeys.byArtist(mbid) });
    }
    prevSynced.current = loadedSyncedAt;
  }, [loadedSyncedAt, mbid, queryClient]);

  useEffect(() => {
    if (!checking || checkFrom === undefined) return;
    const id = setTimeout(() => setCheckStuck(true), CHECK_TIMEOUT_MS);
    return () => clearTimeout(id);
  }, [checking, checkFrom]);

  const notSubscribed = subscription.error instanceof ApiError && subscription.error.status === 404;
  const artist = subscription.data?.artist;

  const records: RecordModel[] = (releases.data ?? [])
    .filter((r) => type === 'all' || r.primaryType === type)
    .map((r) => ({
      mbid: r.mbid,
      title: r.title,
      credit: r.artistCredit,
      date: formatPartialDate(r.firstReleaseDate, t.common),
      coverUrl: r.coverUrl,
      releasePath: `/releases/${r.mbid}`,
      ratingText: r.rating ? formatRating(r.rating, t.locale) : null,
      typeLabel: r.primaryType ? t.release.primaryTypes[r.primaryType] : null,
      secondaryTypes: r.secondaryTypes.map((s) => t.release.secondaryTypes[s] ?? s),
      featured: r.creditRole === 'featured',
      announcement: r.isAnnouncement,
      fresh: !r.isBaseline,
    }));

  return {
    name: artist?.name ?? '',
    details: artist
      ? [artist.disambiguation, artistTypeLabel(artist.type, t.common), artist.country]
          .filter(Boolean)
          .join(' · ')
      : '',
    isLoading: subscription.isPending,
    notSubscribed,
    synced: lastSyncedAt !== null,
    syncText: lastSyncedAt
      ? t.common.checkedAt(formatDateTime(lastSyncedAt, t.locale))
      : t.common.awaitingFirstCheck,
    roleOptions: ROLE_FILTERS.map((value) => ({
      value,
      label: value === 'all' ? t.common.all : t.release.roles[value],
    })),
    typeOptions: TYPE_FILTERS.map((value) => ({
      value,
      label: value === 'all' ? t.common.all : t.release.primaryTypes[value],
    })),
    checking,
    checkStuck,
    checkFailed: requestSync.isError,
    role,
    type,
    /** Some filter hides part of the catalog. */
    filtered: role !== 'all' || type !== 'all',
    records,
    recordsLoading: releases.isPending,
    /** Releases shown under the current filters. */
    count: records.length,

    onRoleChange: setRole,
    onTypeChange: setType,
    onCheckNow: () => {
      setCheckStuck(false);
      setCheckFrom(lastSyncedAt);
      requestSync.mutate(mbid, { onError: () => setCheckFrom(undefined) });
    },
  };
}

export type ArtistCatalogViewModel = ReturnType<typeof useArtistCatalogController>;
