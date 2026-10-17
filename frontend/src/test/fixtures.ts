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

export function syncRunWire(done: number, total: number, finished: boolean, newReleases = 0) {
  return {
    id: 'run-1',
    trigger: 'manual',
    started_at: '2026-10-17T06:00:00Z',
    finished_at: finished ? '2026-10-17T06:05:00Z' : null,
    artists_total: total,
    artists_done: done,
    new_releases: newReleases,
  };
}

export function releaseGroupWire(
  mbid: string,
  title: string,
  role: 'main' | 'collaboration' | 'featured' = 'main',
  extra: Record<string, unknown> = {},
) {
  return {
    mbid,
    title,
    artist_credit: 'Sigur Rós',
    credit_role: role,
    primary_type: 'album',
    secondary_types: [],
    first_release_date: '1999-06',
    is_baseline: true,
    is_announcement: false,
    cover_url: `https://coverartarchive.org/release-group/${mbid}/front-250`,
    musicbrainz_url: `https://musicbrainz.org/release-group/${mbid}`,
    discovered_at: '2026-10-17T06:00:00Z',
    ...extra,
  };
}

export function settingsWire(
  overrides: { emailEnabled?: boolean; to?: string; next?: string | null } = {},
) {
  return {
    filters: {
      primary_types: ['album', 'ep', 'single'],
      excluded_secondary_types: ['Compilation', 'DJ-mix'],
      include_featured: false,
      notify_announcements: true,
    },
    email: {
      enabled: overrides.emailEnabled ?? false,
      to: overrides.to ?? '',
      digest: { frequency: 'daily', weekday: 5, time: '09:00', timezone: 'UTC' },
      last_digest_at: null,
      next_digest_at: overrides.next ?? null,
    },
    rss: { enabled: true, feed_url: 'http://localhost:5173/feeds/tok.atom' },
  };
}

export function feedReleaseWire(n: number, extra: Record<string, unknown> = {}) {
  const mbid = `00000000-0000-0000-0000-${String(n).padStart(12, '0')}`;
  return {
    id: `10000000-0000-0000-0000-${String(n).padStart(12, '0')}`,
    mbid,
    title: `Release ${n}`,
    artist_credit: 'Sigur Rós',
    found_via: {
      mbid: 'f6f2326f-6b25-4170-b89d-e235b25508e8',
      name: 'Sigur Rós',
      credit_role: 'main',
    },
    primary_type: 'album',
    secondary_types: [],
    first_release_date: '2026-10-10',
    is_announcement: false,
    cover_url: `https://coverartarchive.org/release-group/${mbid}/front-250`,
    musicbrainz_url: `https://musicbrainz.org/release-group/${mbid}`,
    discovered_at: '2026-10-17T06:03:00Z',
    ...extra,
  };
}

type ImportCounts = Partial<
  Record<'pending' | 'matched' | 'needs_review' | 'not_found' | 'skipped' | 'failed', number>
>;

/** An import whose total is the sum of the given counts; finished when nothing is pending. */
export function importJobWire(counts: ImportCounts, id = 'job-1', filename = 'artists.txt') {
  const all = {
    pending: 0,
    matched: 0,
    needs_review: 0,
    not_found: 0,
    skipped: 0,
    failed: 0,
    ...counts,
  };
  const total = Object.values(all).reduce((a, b) => a + b, 0);
  return {
    id,
    filename,
    total,
    counts: all,
    finished: all.pending === 0,
    created_at: '2026-10-17T06:00:00Z',
    finished_at: all.pending === 0 ? '2026-10-17T06:02:00Z' : null,
  };
}

/** A needs_review line with two namesakes. */
export function importItemWire(id: string, raw: string, extra: Record<string, unknown> = {}) {
  const candidate = (mbid: string, disambiguation: string, country: string, begin: string) => ({
    mbid,
    name: raw,
    sort_name: raw,
    disambiguation,
    type: 'Group',
    country,
    life_span: { begin, end: null, ended: false },
    score: 100,
  });
  return {
    id,
    line_no: 2,
    raw_value: raw,
    input_mbid: null,
    include_featured: null,
    status: 'needs_review',
    candidates: [
      candidate('5b11f4ce-a62d-471e-81fc-a69a8278c7da', '90s US grunge band', 'US', '1987'),
      candidate('9282c8b4-ca0b-4c6b-b7e3-4f7762dfc4d6', '60s band from the UK', 'GB', '1965'),
    ],
    artist: null,
    error: null,
    ...extra,
  };
}
