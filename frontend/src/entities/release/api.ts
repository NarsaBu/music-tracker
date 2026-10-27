import { z } from 'zod';
import type { HttpClient } from '@/shared/api';
import {
  CREDIT_ROLES,
  PRIMARY_TYPES,
  type CreditRole,
  type CoverInfo,
  type ReleaseDetail,
  type ReleaseGroup,
  type ReleaseInfo,
  type Reviews,
} from './model';

export const ratingSchema = z.object({ value: z.number(), votes: z.number().int() });

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
    rating: ratingSchema.nullable(),
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
    rating: r.rating,
    discoveredAt: r.discovered_at,
  }));

const listSchema = z.object({ items: z.array(releaseGroupSchema) });

const detailSchema = z
  .object({
    mbid: z.string(),
    title: z.string(),
    artist_credit: z.string(),
    primary_type: z.enum(PRIMARY_TYPES).nullable(),
    secondary_types: z.array(z.string()),
    first_release_date: z.string().nullable(),
    is_announcement: z.boolean(),
    cover_url: z.string(),
    musicbrainz_url: z.string(),
    rating: ratingSchema.nullable(),
  })
  .transform((r): ReleaseDetail => ({
    mbid: r.mbid,
    title: r.title,
    artistCredit: r.artist_credit,
    primaryType: r.primary_type,
    secondaryTypes: r.secondary_types,
    firstReleaseDate: r.first_release_date,
    isAnnouncement: r.is_announcement,
    coverUrl: r.cover_url,
    musicbrainzUrl: r.musicbrainz_url,
    rating: r.rating,
  }));

const creditedSchema = z.object({ mbid: z.string(), name: z.string(), subscribed: z.boolean() });

const infoSchema = z
  .object({
    genres: z.array(z.string()),
    edition: z
      .object({
        mbid: z.string(),
        title: z.string(),
        date: z.string().nullable(),
        country: z.string(),
        status: z.string(),
        musicbrainz_url: z.string(),
      })
      .nullable(),
    media: z.array(
      z.object({
        position: z.number().int(),
        format: z.string(),
        title: z.string(),
        tracks: z.array(
          z.object({
            number: z.string(),
            title: z.string(),
            length_ms: z.number().nullable(),
            artist_credit: z.string(),
          }),
        ),
      }),
    ),
    artists: z.array(creditedSchema),
    guests: z.array(creditedSchema),
  })
  .transform((r): ReleaseInfo => ({
    genres: r.genres,
    edition: r.edition && {
      mbid: r.edition.mbid,
      title: r.edition.title,
      date: r.edition.date,
      country: r.edition.country,
      status: r.edition.status,
      musicbrainzUrl: r.edition.musicbrainz_url,
    },
    media: r.media.map((m) => ({
      ...m,
      tracks: m.tracks.map((t) => ({
        number: t.number,
        title: t.title,
        lengthMs: t.length_ms,
        artistCredit: t.artist_credit,
      })),
    })),
    artists: r.artists,
    guests: r.guests,
  }));

const coverInfoSchema = z
  .object({
    available: z.boolean(),
    min_size: z.number().int(),
    width: z.number().int().nullable(),
    height: z.number().int().nullable(),
    format: z.string().nullable(),
    size_bytes: z.number().nullable(),
    filename: z.string().nullable(),
    largest: z.object({ width: z.number().int(), height: z.number().int() }).nullable(),
  })
  .transform((r): CoverInfo => {
    if (r.available && r.width && r.height && r.format && r.filename) {
      return {
        available: true,
        minSize: r.min_size,
        width: r.width,
        height: r.height,
        format: r.format,
        sizeBytes: r.size_bytes,
        filename: r.filename,
      };
    }
    return { available: false, minSize: r.min_size, largest: r.largest };
  });

const reviewsSchema = z
  .object({
    items: z.array(
      z.object({
        id: z.string(),
        author: z.string(),
        rating: z.number().int().nullable(),
        text: z.string(),
        language: z.string(),
        published_at: z.string().nullable(),
        license: z.string(),
        url: z.string(),
      }),
    ),
    total: z.number().int(),
    average: ratingSchema.nullable(),
    url: z.string(),
  })
  .transform((r): Reviews => ({
    ...r,
    items: r.items.map(({ published_at, ...rest }) => ({ ...rest, publishedAt: published_at })),
  }));

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
    get: (mbid: string): Promise<ReleaseDetail> =>
      http.get(`/api/v1/release-groups/${encodeURIComponent(mbid)}`, detailSchema),
    coverInfo: (mbid: string): Promise<CoverInfo> =>
      http.get(`/api/v1/release-groups/${encodeURIComponent(mbid)}/cover/info`, coverInfoSchema),
    info: (mbid: string): Promise<ReleaseInfo> =>
      http.get(`/api/v1/release-groups/${encodeURIComponent(mbid)}/details`, infoSchema),
    reviews: (mbid: string): Promise<Reviews> =>
      http.get(`/api/v1/release-groups/${encodeURIComponent(mbid)}/reviews`, reviewsSchema),
  };
}
