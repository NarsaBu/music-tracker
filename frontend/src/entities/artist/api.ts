import { z } from 'zod';
import type { HttpClient } from '@/shared/api';
import type { ArtistCandidate } from './model';

export const artistSchema = z
  .object({
    mbid: z.string(),
    name: z.string(),
    sort_name: z.string(),
    disambiguation: z.string(),
    type: z.string().nullable(),
    country: z.string().nullable(),
  })
  .transform(({ sort_name, ...rest }) => ({ ...rest, sortName: sort_name }));

const candidateSchema = z
  .object({
    mbid: z.string(),
    name: z.string(),
    sort_name: z.string(),
    disambiguation: z.string(),
    type: z.string().nullable(),
    country: z.string().nullable(),
    life_span: z.object({
      begin: z.string().nullable(),
      end: z.string().nullable(),
      ended: z.boolean(),
    }),
    score: z.number(),
    subscribed: z.boolean(),
  })
  .transform(({ sort_name, life_span, ...rest }): ArtistCandidate => ({
    ...rest,
    sortName: sort_name,
    lifeSpan: life_span,
  }));

const searchResponseSchema = z.object({ items: z.array(candidateSchema) });

export function createArtistApi(http: HttpClient) {
  return {
    search: async (query: string, limit = 10): Promise<ArtistCandidate[]> => {
      const params = new URLSearchParams({ q: query, limit: String(limit) });
      const res = await http.get(`/api/v1/catalog/artists?${params}`, searchResponseSchema);
      return res.items;
    },
  };
}

export type ArtistApi = ReturnType<typeof createArtistApi>;
