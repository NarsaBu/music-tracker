import { http, HttpResponse } from 'msw';
import { httpClient } from '@/shared/api';
import { settingsWire } from '@/test/fixtures';
import { server } from '@/test/server';
import { createSettingsApi } from './api';

const api = createSettingsApi(httpClient);

describe('settings api', () => {
  it('round-trips the wire format', async () => {
    const wire = settingsWire({
      emailEnabled: true,
      to: 'me@example.com',
      next: '2026-10-18T09:00:00Z',
    });
    // MusicBrainz may grow secondary types the UI does not know yet.
    wire.filters.excluded_secondary_types.push('Hologram');
    let body: unknown;
    server.use(
      http.get('/api/v1/settings', () => HttpResponse.json(wire)),
      http.put('/api/v1/settings', async ({ request }) => {
        body = await request.json();
        return HttpResponse.json(wire);
      }),
    );

    const state = await api.get();
    expect(state.nextDigestAt).toBe('2026-10-18T09:00:00Z');
    expect(state.settings.filters.excludedSecondaryTypes).toEqual(['Compilation', 'DJ-mix']);
    expect(state.settings.rss.feedUrl).toMatch(/tok\.atom$/);

    await api.save(state.settings);
    expect(body).toEqual({
      filters: {
        primary_types: ['album', 'ep', 'single'],
        excluded_secondary_types: ['Compilation', 'DJ-mix'],
        include_featured: false,
        notify_announcements: true,
      },
      email: {
        enabled: true,
        to: 'me@example.com',
        digest: { frequency: 'daily', weekday: 5, time: '09:00', timezone: 'UTC' },
      },
      rss: { enabled: true },
    });
  });
});
