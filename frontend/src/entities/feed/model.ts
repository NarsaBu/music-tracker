import type { CreditRole, PrimaryType, Rating } from '@/entities/release/model';

/** A found release in the feed (US-9). */
export interface FeedRelease {
  id: string;
  mbid: string;
  title: string;
  artistCredit: string;
  /** The subscribed artist through whom the release was found. */
  foundVia: { mbid: string; name: string; creditRole: CreditRole };
  primaryType: PrimaryType | null;
  secondaryTypes: string[];
  firstReleaseDate: string | null;
  isAnnouncement: boolean;
  coverUrl: string;
  musicbrainzUrl: string;
  rating: Rating | null;
  discoveredAt: string;
}

/** Feed order, newest first: by when it was found or by release date. */
export type FeedSort = 'found' | 'released';
export const FEED_SORTS: readonly FeedSort[] = ['found', 'released'];

export interface FeedFilter {
  type: PrimaryType | null;
  role: CreditRole | null;
  sort: FeedSort;
}

export interface FeedPage {
  items: FeedRelease[];
  nextCursor: string | null;
}
