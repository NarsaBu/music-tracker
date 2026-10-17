import type { Messages } from '@/shared/i18n';

/** MusicBrainz release group primary types (lowercase, as stored by the backend). */
export const PRIMARY_TYPES = ['album', 'single', 'ep', 'broadcast', 'other'] as const;
export type PrimaryType = (typeof PRIMARY_TYPES)[number];

/** MusicBrainz secondary types, spelled exactly as in the MusicBrainz API. */
export const SECONDARY_TYPES = [
  'Compilation',
  'Soundtrack',
  'Spokenword',
  'Interview',
  'Audiobook',
  'Audio drama',
  'Live',
  'Remix',
  'DJ-mix',
  'Mixtape/Street',
  'Demo',
  'Field recording',
] as const;
export type SecondaryType = (typeof SECONDARY_TYPES)[number];

export const CREDIT_ROLES = ['main', 'collaboration', 'featured'] as const;
export type CreditRole = (typeof CREDIT_ROLES)[number];

/** A release group in an artist's catalog. */
export interface ReleaseGroup {
  mbid: string;
  title: string;
  /** As printed on the cover, e.g. "A feat. B". */
  artistCredit: string;
  /** Role of the catalog's artist in this credit. */
  creditRole: CreditRole;
  primaryType: PrimaryType | null;
  secondaryTypes: string[];
  /** MusicBrainz partial date: "YYYY", "YYYY-MM", "YYYY-MM-DD"; null if unknown. */
  firstReleaseDate: string | null;
  /** Found by the first sync of the artist: history, not news. */
  isBaseline: boolean;
  /** Release date in the future. */
  isAnnouncement: boolean;
  coverUrl: string;
  musicbrainzUrl: string;
  discoveredAt: string;
}

/** "2024", "мая 2024", "17 мая 2024", or "дата неизвестна" (in the UI language). */
export function formatPartialDate(
  date: string | null,
  t: Pick<Messages['common'], 'months' | 'unknownDate'>,
): string {
  if (!date) return t.unknownDate;
  const [y, m, d] = date.split('-');
  const month = m ? t.months[Number(m) - 1] : undefined;
  if (!month) return y ?? date;
  return d ? `${Number(d)} ${month} ${y}` : `${month} ${y}`;
}
