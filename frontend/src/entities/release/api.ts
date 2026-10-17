import { z } from 'zod';
import type { HttpClient } from '@/shared/api';
import { CREDIT_ROLES, PRIMARY_TYPES, type CreditRole, type ReleaseGroup } from './model';

const releaseGroupSchema = z
  .object({
    mbid: z.string(),
    title: z.string(),
    artist_credit: z.string(),
    credit_role: z.enum(CREDIT_ROLES),
    primary_type: z.enum(PRIMARY_TYPES).nullable(),
    secondary_types: z.array(z.string()),
    first_release_date: z.string().nullable(),
    is_baseline: z.boolean(),
    is_announcement: z.boolean(),
    cover_url: z.string(),
    musicbrainz_url: z.string(),
    discovered_at: z.string(),
  })
  .transform((r): ReleaseGroup => ({
    mbid: r.mbid,
    title: r.title,
    artistCredit: r.artist_credit,
    creditRole: r.credit_role,
    primaryType: r.primary_type,
    secondaryTypes: r.secondary_types,
    firstReleaseDate: r.first_release_date,
    isBaseline: r.is_baseline,
    isAnnouncement: r.is_announcement,
    coverUrl: r.cover_url,
    musicbrainzUrl: r.musicbrainz_url,
    discoveredAt: r.discovered_at,
  }));

const listSchema = z.object({ items: z.array(releaseGroupSchema) });

export function createReleaseApi(http: HttpClient) {
  return {
    /** The stored catalog of an artist; role null = every role. */
    listByArtist: async (mbid: string, role: CreditRole | null): Promise<ReleaseGroup[]> => {
      const params = role ? `?${new URLSearchParams({ role })}` : '';
      const res = await http.get(
        `/api/v1/artists/${encodeURIComponent(mbid)}/release-groups${params}`,
        listSchema,
      );
      return res.items;
    },
  };
}
