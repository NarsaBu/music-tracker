import { z } from 'zod';
import { artistSchema } from '@/entities/artist/api';
import { PRIMARY_TYPES } from '@/entities/release/model';
import type { HttpClient } from '@/shared/api';
import type {
  FilterPatch,
  SubscribeRequest,
  Subscription,
  SubscriptionListParams,
  SubscriptionPage,
} from './model';

const overridable = <T extends z.ZodType>(value: T) => z.object({ value, overridden: z.boolean() });

const subscriptionSchema = z
  .object({
    artist: artistSchema,
    filter: z.object({
      primary_types: overridable(z.array(z.enum(PRIMARY_TYPES))),
      excluded_secondary_types: overridable(z.array(z.string())),
      include_featured: overridable(z.boolean()),
    }),
    last_synced_at: z.string().nullable(),
    baseline_at: z.string(),
    created_at: z.string(),
  })
  .transform(({ artist, filter, last_synced_at, baseline_at, created_at }): Subscription => ({
    artist,
    filter: {
      primaryTypes: filter.primary_types,
      excludedSecondaryTypes: filter.excluded_secondary_types,
      includeFeatured: filter.include_featured,
    },
    lastSyncedAt: last_synced_at,
    baselineAt: baseline_at,
    createdAt: created_at,
  }));

const pageSchema = z.object({
  items: z.array(subscriptionSchema),
  total: z.number(),
  facets: z
    .object({ countries: z.array(z.string()), types: z.array(z.string()) })
    .default({ countries: [], types: [] }),
});

export function createSubscriptionApi(http: HttpClient) {
  return {
    list: ({
      search,
      limit,
      offset,
      ...filters
    }: SubscriptionListParams): Promise<SubscriptionPage> => {
      const params = new URLSearchParams({ limit: String(limit), offset: String(offset) });
      if (search) params.set('q', search);
      for (const [key, value] of Object.entries(filters)) {
        if (value) params.set(key, value);
      }
      return http.get(`/api/v1/subscriptions?${params}`, pageSchema);
    },
    subscribe: (req: SubscribeRequest): Promise<Subscription> =>
      http.post(
        '/api/v1/subscriptions',
        {
          mbid: req.mbid,
          include_featured: req.includeFeatured,
          primary_types: req.primaryTypes,
          excluded_secondary_types: req.excludedSecondaryTypes,
        },
        subscriptionSchema,
      ),
    /** Changes overrides: a field set to null goes back to the global setting. */
    updateFilter: (mbid: string, patch: FilterPatch): Promise<Subscription> =>
      http.patch(
        `/api/v1/subscriptions/${encodeURIComponent(mbid)}`,
        {
          primary_types: patch.primaryTypes,
          excluded_secondary_types: patch.excludedSecondaryTypes,
          include_featured: patch.includeFeatured,
        },
        subscriptionSchema,
      ),
    get: (mbid: string): Promise<Subscription> =>
      http.get(`/api/v1/subscriptions/${encodeURIComponent(mbid)}`, subscriptionSchema),
    unsubscribe: (mbid: string): Promise<void> =>
      http.delete(`/api/v1/subscriptions/${encodeURIComponent(mbid)}`),
    /** Queues a check of one artist; it runs in the background (202). */
    requestSync: async (mbid: string): Promise<void> => {
      await http.post(
        `/api/v1/subscriptions/${encodeURIComponent(mbid)}/sync`,
        undefined,
        z.undefined(),
      );
    },
  };
}

export type SubscriptionApi = ReturnType<typeof createSubscriptionApi>;
