import { useState } from 'react';
import { useFeedQuery } from '@/entities/feed/queries';
import {
  CREDIT_ROLE_LABELS,
  formatPartialDate,
  PRIMARY_TYPE_LABELS,
  type CreditRole,
  type PrimaryType,
} from '@/entities/release/model';
import { formatDateTime } from '@/shared/lib/format';

export type TypeFilter = PrimaryType | 'all';
export type RoleFilter = CreditRole | 'all';

export const TYPE_OPTIONS: ReadonlyArray<{ value: TypeFilter; label: string }> = [
  { value: 'all', label: 'Все' },
  { value: 'album', label: PRIMARY_TYPE_LABELS.album },
  { value: 'ep', label: PRIMARY_TYPE_LABELS.ep },
  { value: 'single', label: PRIMARY_TYPE_LABELS.single },
  { value: 'other', label: PRIMARY_TYPE_LABELS.other },
];

export const ROLE_OPTIONS: ReadonlyArray<{ value: RoleFilter; label: string }> = [
  { value: 'all', label: 'Все' },
  { value: 'main', label: CREDIT_ROLE_LABELS.main },
  { value: 'collaboration', label: CREDIT_ROLE_LABELS.collaboration },
  { value: 'featured', label: CREDIT_ROLE_LABELS.featured },
];

export interface FeedRecordModel {
  id: string;
  title: string;
  credit: string;
  date: string;
  foundText: string;
  coverUrl: string;
  musicbrainzUrl: string;
  artistMbid: string;
  typeLabel: string | null;
  secondaryTypes: string[];
  featured: boolean;
  announcement: boolean;
}

/** Feed of found releases ("record shelf", US-9). */
export function useReleasesFeedController() {
  const [type, setType] = useState<TypeFilter>('all');
  const [role, setRole] = useState<RoleFilter>('all');
  const feed = useFeedQuery({
    type: type === 'all' ? null : type,
    role: role === 'all' ? null : role,
  });

  const releases = feed.data?.pages.flatMap((p) => p.items) ?? [];
  const records: FeedRecordModel[] = releases.map((r) => ({
    id: r.id,
    title: r.title,
    credit: r.artistCredit,
    date: formatPartialDate(r.firstReleaseDate),
    foundText:
      r.foundVia.creditRole === 'featured'
        ? `Гость: ${r.foundVia.name} · найдено ${formatDateTime(r.discoveredAt)}`
        : `Найдено ${formatDateTime(r.discoveredAt)}`,
    coverUrl: r.coverUrl,
    musicbrainzUrl: r.musicbrainzUrl,
    artistMbid: r.foundVia.mbid,
    typeLabel: r.primaryType ? PRIMARY_TYPE_LABELS[r.primaryType] : null,
    secondaryTypes: r.secondaryTypes,
    featured: r.foundVia.creditRole === 'featured',
    announcement: r.isAnnouncement,
  }));
  const filtered = type !== 'all' || role !== 'all';

  return {
    type,
    role,
    records,
    isLoading: feed.isPending,
    isError: feed.isError,
    isEmpty: feed.isSuccess && records.length === 0,
    filtered,
    lcdText: feed.isPending
      ? '---'
      : `${String(records.length).padStart(3, '0')}${feed.hasNextPage ? '+' : ''}`,
    hasMore: feed.hasNextPage,
    loadingMore: feed.isFetchingNextPage,

    onTypeChange: setType,
    onRoleChange: setRole,
    onLoadMore: () => void feed.fetchNextPage(),
  };
}

export type ReleasesFeedViewModel = ReturnType<typeof useReleasesFeedController>;
