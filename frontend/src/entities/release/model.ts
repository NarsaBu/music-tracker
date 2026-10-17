/** MusicBrainz release group primary types (lowercase, as stored by the backend). */
export const PRIMARY_TYPES = ['album', 'single', 'ep', 'broadcast', 'other'] as const;
export type PrimaryType = (typeof PRIMARY_TYPES)[number];

export const PRIMARY_TYPE_LABELS: Record<PrimaryType, string> = {
  album: 'Album',
  single: 'Single',
  ep: 'EP',
  broadcast: 'Broadcast',
  other: 'Other',
};

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

export const CREDIT_ROLE_LABELS: Record<CreditRole, string> = {
  main: 'Основные',
  collaboration: 'Совместные',
  featured: 'Featuring',
};

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

const MONTHS = ['янв', 'фев', 'мар', 'апр', 'мая', 'июн', 'июл', 'авг', 'сен', 'окт', 'ноя', 'дек'];

/** "2024", "мая 2024", "17 мая 2024", or "дата неизвестна". */
export function formatPartialDate(date: string | null): string {
  if (!date) return 'дата неизвестна';
  const [y, m, d] = date.split('-');
  const month = m ? MONTHS[Number(m) - 1] : undefined;
  if (!month) return y ?? date;
  return d ? `${Number(d)} ${month} ${y}` : `${month} ${y}`;
}
