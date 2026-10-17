import type { Messages } from '@/shared/i18n';

/** MusicBrainz partial dates: "YYYY", "YYYY-MM" or "YYYY-MM-DD". */
export interface LifeSpan {
  begin: string | null;
  end: string | null;
  ended: boolean;
}

export interface Artist {
  mbid: string;
  name: string;
  sortName: string;
  disambiguation: string;
  /** Person, Group, …; null when MusicBrainz does not know it. */
  type: string | null;
  /** ISO 3166-1 alpha-2. */
  country: string | null;
}

/** An artist search result ("station" on the tuner). */
export interface ArtistCandidate extends Artist {
  lifeSpan: LifeSpan;
  /** MusicBrainz relevance, 0–100. */
  score: number;
  subscribed: boolean;
}

/** "1987–1994", "1994–", "?–1994" or "" — years only, enough to tell namesakes apart. */
export function formatLifeSpan({ begin, end, ended }: LifeSpan): string {
  const from = begin?.slice(0, 4) ?? '';
  const to = end?.slice(0, 4) ?? (ended ? '?' : '');
  if (!from && !to) return '';
  return `${from || '?'}–${to}`;
}

/** MusicBrainz artist type ("Person", "Group", …) in the UI language. */
export function artistTypeLabel(
  type: string | null | undefined,
  t: Pick<Messages['common'], 'artistTypes'>,
): string | null {
  return type ? (t.artistTypes[type] ?? type) : null;
}
