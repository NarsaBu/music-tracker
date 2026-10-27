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
  /** MusicBrainz community rating; null when nobody voted. */
  rating: Rating | null;
  discoveredAt: string;
}

/** One release on its own page: what the tracker stores about it. */
export interface ReleaseDetail {
  mbid: string;
  title: string;
  artistCredit: string;
  primaryType: PrimaryType | null;
  secondaryTypes: string[];
  firstReleaseDate: string | null;
  isAnnouncement: boolean;
  coverUrl: string;
  musicbrainzUrl: string;
  rating: Rating | null;
}

/** An artist credited on a release; links to the artist page when subscribed. */
export interface CreditedArtist {
  mbid: string;
  name: string;
  subscribed: boolean;
}

export interface Track {
  /** As printed: "1", "A1", … */
  number: string;
  title: string;
  /** null when MusicBrainz does not know it. */
  lengthMs: number | null;
  artistCredit: string;
}

export interface Medium {
  position: number;
  /** CD, Digital Media, 12" Vinyl, …; empty when unknown. */
  format: string;
  title: string;
  tracks: Track[];
}

/** Release page details from MusicBrainz: genres, tracklist, credited artists. */
export interface ReleaseInfo {
  /** Most voted first. */
  genres: string[];
  /** The edition the tracklist comes from: the earliest official one; null before release. */
  edition: {
    mbid: string;
    title: string;
    date: string | null;
    country: string;
    status: string;
    musicbrainzUrl: string;
  } | null;
  media: Medium[];
  artists: CreditedArtist[];
  /** Credited on tracks but not on the release: featurings. */
  guests: CreditedArtist[];
}

/** The artist page in the app when subscribed, otherwise MusicBrainz. */
export function artistLink(a: CreditedArtist): { to: string } | { href: string } {
  return a.subscribed
    ? { to: `/artists/${a.mbid}` }
    : { href: `https://musicbrainz.org/artist/${a.mbid}` };
}

/** "3:05", "1:02:09". */
export function formatLength(ms: number): string {
  const total = Math.round(ms / 1000);
  const h = Math.floor(total / 3600);
  const m = Math.floor((total % 3600) / 60);
  const s = String(total % 60).padStart(2, '0');
  return h > 0 ? `${h}:${String(m).padStart(2, '0')}:${s}` : `${m}:${s}`;
}

/** A user review from CritiqueBrainz (Creative Commons text). */
export interface Review {
  id: string;
  author: string;
  /** 1–5; null when the author gave no rating. */
  rating: number | null;
  text: string;
  language: string;
  publishedAt: string | null;
  license: string;
  url: string;
}

export interface Reviews {
  items: Review[];
  /** All reviews, items holds the most popular ones. */
  total: number;
  /** Mean of the rated reviews; null when none is rated. */
  average: Rating | null;
  /** All reviews of the release on CritiqueBrainz. */
  url: string;
}

/** Average of user votes on a 0–5 scale. */
export interface Rating {
  value: number;
  votes: number;
}

const ratingFormats = new Map<string, Intl.NumberFormat>();

/** "★ 4,6 · 89" (ru-RU): one decimal, in the UI locale. */
export function formatRating({ value, votes }: Rating, locale: string): string {
  let f = ratingFormats.get(locale);
  if (!f) {
    f = new Intl.NumberFormat(locale, { minimumFractionDigits: 1, maximumFractionDigits: 1 });
    ratingFormats.set(locale, f);
  }
  return `★ ${f.format(value)} · ${votes}`;
}

/** Search pages of sites that rate releases but have no API to read the scores from. */
export function ratingSearchLinks(artistCredit: string, title: string) {
  const q = `${artistCredit} ${title}`;
  return {
    albumOfTheYear: `https://www.albumoftheyear.org/search/?q=${encodeURIComponent(q)}`,
    rateYourMusic: `https://rateyourmusic.com/search?searchtype=l&searchterm=${encodeURIComponent(q)}`,
  };
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
