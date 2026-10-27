import { z } from 'zod';
import { CREDIT_ROLES, PRIMARY_TYPES } from '@/entities/release/model';
import type { HttpClient } from '@/shared/api';
import type { FeedFilter, FeedPage, FeedRelease } from './model';

const releaseSchema = z
  .object({
    id: z.string(),
    mbid: z.string(),
    title: z.string(),
    artist_credit: z.string(),
    found_via: z.object({ mbid: z.string(), name: z.string(), credit_role: z.enum(CREDIT_ROLES) }),
    primary_type: z.enum(PRIMARY_TYPES).nullable(),
    secondary_types: z.array(z.string()),
    first_release_date: z.string().nullable(),
    is_announcement: z.boolean(),
    cover_url: z.string(),
    musicbrainz_url: z.string(),
    discovered_at: z.string(),
  })
  .transform((r): FeedRelease => ({
    id: r.id,
    mbid: r.mbid,
    title: r.title,
    artistCredit: r.artist_credit,
    foundVia: {
      mbid: r.found_via.mbid,
      name: r.found_via.name,
      creditRole: r.found_via.credit_role,
    },
    primaryType: r.primary_type,
    secondaryTypes: r.secondary_types,
    firstReleaseDate: r.first_release_date,
    isAnnouncement: r.is_announcement,
    coverUrl: r.cover_url,
    musicbrainzUrl: r.musicbrainz_url,
    discoveredAt: r.discovered_at,
  }));

const pageSchema = z
  .object({ items: z.array(releaseSchema), next_cursor: z.string().nullable() })
  .transform((p): FeedPage => ({ items: p.items, nextCursor: p.next_cursor }));

export function createFeedApi(http: HttpClient) {
  return {
    page: (filter: FeedFilter, cursor: string | null, limit: number): Promise<FeedPage> => {
      const params = new URLSearchParams({ limit: String(limit) });
      if (cursor) params.set('cursor', cursor);
      if (filter.type) params.set('type', filter.type);
      if (filter.role) params.set('role', filter.role);
      if (filter.sort !== 'found') params.set('sort', filter.sort);
      return http.get(`/api/v1/releases?${params}`, pageSchema);
    },
  };
}
