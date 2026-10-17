import type { CreditRole, PrimaryType } from '@/entities/release/model';

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
  discoveredAt: string;
}

export interface FeedFilter {
  type: PrimaryType | null;
  role: CreditRole | null;
}

export interface FeedPage {
  items: FeedRelease[];
  nextCursor: string | null;
}
