import { useQueryClient } from '@tanstack/react-query';
import { useEffect, useRef, useState } from 'react';
import {
  CREDIT_ROLE_LABELS,
  formatPartialDate,
  PRIMARY_TYPE_LABELS,
  type CreditRole,
} from '@/entities/release/model';
import { releaseKeys, useArtistReleaseGroupsQuery } from '@/entities/release/queries';
import { useRequestSyncMutation, useSubscriptionQuery } from '@/entities/subscription/queries';
import { ApiError } from '@/shared/api';
import { formatDateTime } from '@/shared/lib/format';

export type RoleFilter = CreditRole | 'all';

export const ROLE_OPTIONS: ReadonlyArray<{ value: RoleFilter; label: string }> = [
  { value: 'all', label: 'Все' },
  { value: 'main', label: CREDIT_ROLE_LABELS.main },
  { value: 'collaboration', label: CREDIT_ROLE_LABELS.collaboration },
  { value: 'featured', label: CREDIT_ROLE_LABELS.featured },
];

export const CHECK_POLL_MS = 2_000;
/** A requested check that has not landed by then is reported as stuck (the job keeps retrying). */
export const CHECK_TIMEOUT_MS = 3 * 60_000;

export interface RecordModel {
  mbid: string;
  title: string;
  credit: string;
  date: string;
  coverUrl: string;
  musicbrainzUrl: string;
  typeLabel: string | null;
  secondaryTypes: string[];
  featured: boolean;
  announcement: boolean;
  /** Discovered after the baseline: this is what notifications are about. */
  fresh: boolean;
}

/** Artist page (US-10): stored catalog, role filter, "check now". */
export function useArtistCatalogController(mbid: string) {
  const queryClient = useQueryClient();
  const [role, setRole] = useState<RoleFilter>('all');
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

  const records: RecordModel[] = (releases.data ?? []).map((r) => ({
    mbid: r.mbid,
    title: r.title,
    credit: r.artistCredit,
    date: formatPartialDate(r.firstReleaseDate),
    coverUrl: r.coverUrl,
    musicbrainzUrl: r.musicbrainzUrl,
    typeLabel: r.primaryType ? PRIMARY_TYPE_LABELS[r.primaryType] : null,
    secondaryTypes: r.secondaryTypes,
    featured: r.creditRole === 'featured',
    announcement: r.isAnnouncement,
    fresh: !r.isBaseline,
  }));

  return {
    name: artist?.name ?? '',
    details: artist
      ? [artist.disambiguation, artist.type, artist.country].filter(Boolean).join(' · ')
      : '',
    isLoading: subscription.isPending,
    notSubscribed,
    synced: lastSyncedAt !== null,
    syncText: lastSyncedAt ? `Проверен ${formatDateTime(lastSyncedAt)}` : 'Ждёт первой проверки',
    checking,
    checkStuck,
    checkFailed: requestSync.isError,
    role,
    records,
    recordsLoading: releases.isPending,
    count: releases.data?.length ?? 0,

    onRoleChange: setRole,
    onCheckNow: () => {
      setCheckStuck(false);
      setCheckFrom(lastSyncedAt);
      requestSync.mutate(mbid, { onError: () => setCheckFrom(undefined) });
    },
  };
}

export type ArtistCatalogViewModel = ReturnType<typeof useArtistCatalogController>;
