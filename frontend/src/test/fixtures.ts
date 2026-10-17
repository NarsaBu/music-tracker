/** Wire-format (snake_case) API payloads shared by tests. */

export const sigurRosCandidate = {
  mbid: 'f6f2326f-6b25-4170-b89d-e235b25508e8',
  name: 'Sigur Rós',
  sort_name: 'Sigur Rós',
  disambiguation: 'Icelandic post‐rock band',
  type: 'Group',
  country: 'IS',
  life_span: { begin: '1994-01-04', end: null, ended: false },
  score: 100,
  subscribed: false,
};

export function subscriptionWire(mbid: string, name: string, lastSyncedAt: string | null = null) {
  return {
    artist: { mbid, name, sort_name: name, disambiguation: '', type: 'Group', country: 'IS' },
    filter: {
      primary_types: { value: ['album', 'ep'], overridden: true },
      excluded_secondary_types: { value: ['Compilation'], overridden: false },
      include_featured: { value: false, overridden: false },
    },
    last_synced_at: lastSyncedAt,
    baseline_at: '2026-10-17T06:00:00Z',
    created_at: '2026-10-17T06:00:00Z',
  };
}
