import { act, renderHook, waitFor } from '@testing-library/react';
import { http, HttpResponse } from 'msw';
import { subscriptionWire } from '@/test/fixtures';
import { createWrapper } from '@/test/queryWrapper';
import { server } from '@/test/server';
import { useSimilarArtistsController } from './useSimilarArtistsController';

const ARTIST = '329f95db-f02e-40bf-a5f7-0dea11529ce3';
const BMTH = '074e3847-f67f-49f9-81f1-8c8cea147e8e';
const LP = 'f59c5520-5f46-4d2c-b2c4-822eabf53419';

function renderController() {
  const { Wrapper } = createWrapper();
  return renderHook(() => useSimilarArtistsController(ARTIST), { wrapper: Wrapper });
}

describe('useSimilarArtistsController', () => {
  it('scales the match to the closest artist and subscribes in place', async () => {
    let subscribed: unknown;
    server.use(
      http.get(`/api/v1/artists/${ARTIST}/similar`, () =>
        HttpResponse.json({
          items: [
            {
              mbid: BMTH,
              name: 'Bring Me the Horizon',
              disambiguation: '',
              type: 'Group',
              score: 1200,
              subscribed: true,
            },
            {
              mbid: LP,
              name: 'Linkin Park',
              disambiguation: 'American rock band',
              type: null,
              score: 300,
              subscribed: false,
            },
          ],
        }),
      ),
      http.post('/api/v1/subscriptions', async ({ request }) => {
        subscribed = await request.json();
        return HttpResponse.json(subscriptionWire(LP, 'Linkin Park'), { status: 201 });
      }),
    );
    const { result } = renderController();

    await waitFor(() => expect(result.current.rows).toHaveLength(2));
    expect(result.current.rows[0]).toMatchObject({
      match: 100,
      subscribed: true,
      details: 'Группа',
    });
    expect(result.current.rows[1]).toMatchObject({
      match: 25,
      subscribed: false,
      details: 'American rock band',
    });

    act(() => result.current.onSubscribe(LP));
    await waitFor(() => expect(result.current.rows[1]?.subscribed).toBe(true));
    expect(subscribed).toMatchObject({ mbid: LP });
  });

  it('tells an unknown artist from ListenBrainz being down', async () => {
    server.use(
      http.get(`/api/v1/artists/${ARTIST}/similar`, () => HttpResponse.json({ items: [] })),
    );
    const empty = renderController();
    await waitFor(() => expect(empty.result.current.isEmpty).toBe(true));
    expect(empty.result.current.errorText).toBeUndefined();

    server.use(
      http.get(`/api/v1/artists/${ARTIST}/similar`, () =>
        HttpResponse.json(
          { type: 'about:blank', title: 'Upstream unavailable', status: 503 },
          { status: 503, headers: { 'Content-Type': 'application/problem+json' } },
        ),
      ),
    );
    const down = renderController();
    await waitFor(() => expect(down.result.current.loadFailed).toBe(true));
    expect(down.result.current.errorText).toBe('ListenBrainz не отвечает, попробуйте позже');
  });
});
