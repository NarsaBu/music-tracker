import { z } from 'zod';
import { PRIMARY_TYPES, SECONDARY_TYPES } from '@/entities/release/model';
import type { HttpClient } from '@/shared/api';
import type { Settings, SettingsState, Weekday } from './model';

const settingsSchema = z
  .object({
    filters: z.object({
      primary_types: z.array(z.enum(PRIMARY_TYPES)),
      // Unknown secondary types (MusicBrainz adds new ones) are dropped, not fatal.
      excluded_secondary_types: z.array(z.string()),
      include_featured: z.boolean(),
      notify_announcements: z.boolean(),
    }),
    email: z.object({
      enabled: z.boolean(),
      to: z.string(),
      digest: z.object({
        frequency: z.enum(['daily', 'weekly']),
        weekday: z.number().int().min(0).max(6),
        time: z.string(),
        timezone: z.string(),
      }),
      last_digest_at: z.string().nullable(),
      next_digest_at: z.string().nullable(),
    }),
    rss: z.object({ enabled: z.boolean(), feed_url: z.string() }),
  })
  .transform(({ filters, email, rss }): SettingsState => ({
    settings: {
      filters: {
        primaryTypes: filters.primary_types,
        excludedSecondaryTypes: filters.excluded_secondary_types.filter(
          (t): t is (typeof SECONDARY_TYPES)[number] =>
            (SECONDARY_TYPES as readonly string[]).includes(t),
        ),
        includeFeatured: filters.include_featured,
        notifyAnnouncements: filters.notify_announcements,
      },
      email: {
        enabled: email.enabled,
        to: email.to,
        digest: { ...email.digest, weekday: email.digest.weekday as Weekday },
      },
      rss: { enabled: rss.enabled, feedUrl: rss.feed_url },
    },
    lastDigestAt: email.last_digest_at,
    nextDigestAt: email.next_digest_at,
  }));

function toWire(s: Settings) {
  return {
    filters: {
      primary_types: s.filters.primaryTypes,
      excluded_secondary_types: s.filters.excludedSecondaryTypes,
      include_featured: s.filters.includeFeatured,
      notify_announcements: s.filters.notifyAnnouncements,
    },
    email: { enabled: s.email.enabled, to: s.email.to, digest: s.email.digest },
    rss: { enabled: s.rss.enabled },
  };
}

export function createSettingsApi(http: HttpClient) {
  return {
    get: (): Promise<SettingsState> => http.get('/api/v1/settings', settingsSchema),
    save: (s: Settings): Promise<SettingsState> =>
      http.put('/api/v1/settings', toWire(s), settingsSchema),
    sendTestEmail: async (): Promise<void> => {
      await http.post('/api/v1/settings/email/test', undefined, z.undefined());
    },
    rotateFeedToken: async (): Promise<string> => {
      const res = await http.post(
        '/api/v1/settings/feed-token/rotate',
        undefined,
        z.object({ feed_url: z.string() }),
      );
      return res.feed_url;
    },
  };
}
