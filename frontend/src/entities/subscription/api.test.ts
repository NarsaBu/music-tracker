import { http, HttpResponse } from 'msw';
import { httpClient } from '@/shared/api';
import { server } from '@/test/server';
import { createSubscriptionApi } from './api';

const api = createSubscriptionApi(httpClient);

const wire = {
  artist: {
    mbid: 'f6f2326f-6b25-4170-b89d-e235b25508e8',
    name: 'Sigur Rós',
    sort_name: 'Sigur Rós',
    disambiguation: 'Icelandic post‐rock band',
    type: 'Group',
    country: 'IS',
  },
  filter: {
    primary_types: { value: ['album', 'ep'], overridden: false },
    excluded_secondary_types: { value: ['Compilation'], overridden: false },
    include_featured: { value: true, overridden: true },
  },
  last_synced_at: null,
  baseline_at: '2026-10-17T06:00:00Z',
  created_at: '2026-10-17T06:00:00Z',
};

describe('subscription api', () => {
  it('maps the wire format to the model', async () => {
    let url = '';
    server.use(
      http.get('/api/v1/subscriptions', ({ request }) => {
        url = request.url;
        return HttpResponse.json({ items: [wire], total: 1 });
      }),
    );

    const page = await api.list({ search: 'sig', limit: 50, offset: 0 });

    expect(new URL(url).searchParams.toString()).toBe('limit=50&offset=0&q=sig');
    expect(page.total).toBe(1);
    expect(page.items[0]).toMatchObject({
      artist: { sortName: 'Sigur Rós', country: 'IS' },
      filter: { includeFeatured: { value: true, overridden: true } },
      lastSyncedAt: null,
    });
  });

  it('sends only the overrides that are set', async () => {
    let body: unknown;
    server.use(
      http.post('/api/v1/subscriptions', async ({ request }) => {
        body = await request.json();
        return HttpResponse.json(wire, { status: 201 });
      }),
    );

    await api.subscribe({ mbid: wire.artist.mbid, includeFeatured: true });

    expect(body).toEqual({ mbid: wire.artist.mbid, include_featured: true });
  });
});
