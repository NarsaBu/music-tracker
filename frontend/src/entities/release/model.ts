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

export type CreditRole = 'main' | 'collaboration' | 'featured';
