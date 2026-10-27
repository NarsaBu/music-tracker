import { useState } from 'react';
import { FEED_SORTS, type FeedSort } from '@/entities/feed/model';
import { useFeedQuery } from '@/entities/feed/queries';
import {
  CREDIT_ROLES,
  formatPartialDate,
  type CreditRole,
  type PrimaryType,
} from '@/entities/release/model';
import { useI18n } from '@/shared/i18n';
import { formatDateTime, LCD_BLANK, lcdCount } from '@/shared/lib/format';

export type TypeFilter = PrimaryType | 'all';
export type RoleFilter = CreditRole | 'all';

export const TYPE_FILTERS: readonly TypeFilter[] = ['all', 'album', 'ep', 'single', 'other'];
export const ROLE_FILTERS: readonly RoleFilter[] = ['all', ...CREDIT_ROLES];

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
  const { t } = useI18n();
  const at = (iso: string) => formatDateTime(iso, t.locale);
  const [type, setType] = useState<TypeFilter>('all');
  const [role, setRole] = useState<RoleFilter>('all');
  const [sort, setSort] = useState<FeedSort>('found');
  const feed = useFeedQuery({
    type: type === 'all' ? null : type,
    role: role === 'all' ? null : role,
    sort,
  });

  const releases = feed.data?.pages.flatMap((p) => p.items) ?? [];
  const records: FeedRecordModel[] = releases.map((r) => ({
    id: r.id,
    title: r.title,
    credit: r.artistCredit,
    date: formatPartialDate(r.firstReleaseDate, t.common),
    foundText:
      r.foundVia.creditRole === 'featured'
        ? t.feed.guest(r.foundVia.name, at(r.discoveredAt))
        : t.feed.found(at(r.discoveredAt)),
    coverUrl: r.coverUrl,
    musicbrainzUrl: r.musicbrainzUrl,
    artistMbid: r.foundVia.mbid,
    typeLabel: r.primaryType ? t.release.primaryTypes[r.primaryType] : null,
    secondaryTypes: r.secondaryTypes.map((s) => t.release.secondaryTypes[s] ?? s),
    featured: r.foundVia.creditRole === 'featured',
    announcement: r.isAnnouncement,
  }));
  const filtered = type !== 'all' || role !== 'all';

  return {
    type,
    role,
    sort,
    records,
    isLoading: feed.isPending,
    isError: feed.isError,
    isEmpty: feed.isSuccess && records.length === 0,
    filtered,
    typeOptions: TYPE_FILTERS.map((value) => ({
      value,
      label: value === 'all' ? t.common.all : t.release.primaryTypes[value],
    })),
    roleOptions: ROLE_FILTERS.map((value) => ({
      value,
      label: value === 'all' ? t.common.all : t.release.roles[value],
    })),
    sortOptions: FEED_SORTS.map((value) => ({ value, label: t.feed.sortOptions[value] })),
    lcdText: feed.isPending
      ? LCD_BLANK
      : `${lcdCount(records.length)}${feed.hasNextPage ? '+' : ''}`,
    hasMore: feed.hasNextPage,
    loadingMore: feed.isFetchingNextPage,

    onTypeChange: setType,
    onRoleChange: setRole,
    onSortChange: setSort,
    onLoadMore: () => void feed.fetchNextPage(),
  };
}

export type ReleasesFeedViewModel = ReturnType<typeof useReleasesFeedController>;
