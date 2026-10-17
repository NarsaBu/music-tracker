import { act, renderHook, waitFor } from '@testing-library/react';
import { http, HttpResponse } from 'msw';
import { sigurRosCandidate, subscriptionWire } from '@/test/fixtures';
import { createWrapper } from '@/test/queryWrapper';
import { server } from '@/test/server';
import { useArtistSearchController } from './useArtistSearchController';

function renderController() {
  const { Wrapper } = createWrapper();
  return renderHook(() => useArtistSearchController(), { wrapper: Wrapper });
}

describe('useArtistSearchController', () => {
  it('is idle until something is typed', () => {
    const { result } = renderController();
    expect(result.current.status).toBe('idle');
    expect(result.current.lcdText).toBe('NO SIGNAL');
    expect(result.current.stations).toEqual([]);
  });

  it('debounces typing into a single search request', async () => {
    const queries: string[] = [];
    server.use(
      http.get('/api/v1/catalog/artists', ({ request }) => {
        queries.push(new URL(request.url).searchParams.get('q') ?? '');
        return HttpResponse.json({ items: [sigurRosCandidate] });
      }),
    );
    const { result } = renderController();

    act(() => result.current.onQueryChange('sig'));
    act(() => result.current.onQueryChange('sigur'));
    expect(result.current.status).toBe('tuning');

    await waitFor(() => expect(result.current.status).toBe('results'));
    expect(queries).toEqual(['sigur']);
    expect(result.current.lcdText).toBe('1 STATIONS');
    expect(result.current.stations[0]).toMatchObject({
      name: 'Sigur Rós',
      details: 'Icelandic post‐rock band · Group',
      years: '1994–',
      subscribed: false,
    });
  });

  it('reports an empty result and a MusicBrainz outage', async () => {
    server.use(http.get('/api/v1/catalog/artists', () => HttpResponse.json({ items: [] })));
    const { result } = renderController();

    act(() => result.current.onQueryChange('zzqxj'));
    await waitFor(() => expect(result.current.status).toBe('empty'));

    server.use(
      http.get('/api/v1/catalog/artists', () =>
        HttpResponse.json({ title: 'Upstream unavailable' }, { status: 503 }),
      ),
    );
    act(() => result.current.onQueryChange('zzqxjv'));
    await waitFor(() => expect(result.current.status).toBe('error'));
    expect(result.current.errorText).toMatch(/MusicBrainz не отвечает/);
  });

  it('subscribes with the featuring override and marks the station', async () => {
    let body: unknown;
    server.use(
      http.get('/api/v1/catalog/artists', () => HttpResponse.json({ items: [sigurRosCandidate] })),
      http.post('/api/v1/subscriptions', async ({ request }) => {
        body = await request.json();
        return HttpResponse.json(subscriptionWire(sigurRosCandidate.mbid, 'Sigur Rós'), {
          status: 201,
        });
      }),
    );
    const { result } = renderController();
    act(() => result.current.onQueryChange('sigur'));
    await waitFor(() => expect(result.current.status).toBe('results'));

    act(() => result.current.onIncludeFeaturedChange(true));
    act(() => result.current.onSubscribe(sigurRosCandidate.mbid));

    await waitFor(() => expect(result.current.stations[0]?.subscribed).toBe(true));
    expect(body).toEqual({ mbid: sigurRosCandidate.mbid, include_featured: true });
  });

  it('leaves featuring to the global setting when the switch is off', async () => {
    let body: unknown;
    server.use(
      http.get('/api/v1/catalog/artists', () => HttpResponse.json({ items: [sigurRosCandidate] })),
      http.post('/api/v1/subscriptions', async ({ request }) => {
        body = await request.json();
        return HttpResponse.json(subscriptionWire(sigurRosCandidate.mbid, 'Sigur Rós'));
      }),
    );
    const { result } = renderController();
    act(() => result.current.onQueryChange('sigur'));
    await waitFor(() => expect(result.current.status).toBe('results'));

    act(() => result.current.onSubscribe(sigurRosCandidate.mbid));
    await waitFor(() => expect(body).toEqual({ mbid: sigurRosCandidate.mbid }));
  });
});
