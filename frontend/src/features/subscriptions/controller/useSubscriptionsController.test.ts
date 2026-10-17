import { act, renderHook, waitFor } from '@testing-library/react';
import { http, HttpResponse } from 'msw';
import { subscriptionWire } from '@/test/fixtures';
import { createWrapper } from '@/test/queryWrapper';
import { server } from '@/test/server';
import { PAGE_SIZE, useSubscriptionsController } from './useSubscriptionsController';

const SIGUR = 'f6f2326f-6b25-4170-b89d-e235b25508e8';
const NIRVANA = '5b11f4ce-a62d-471e-81fc-a69a8278c7da';

function renderController(route = '/artists') {
  const { Wrapper } = createWrapper({ route });
  return renderHook(() => useSubscriptionsController(), { wrapper: Wrapper });
}

/** Records the query string of every list request. */
function recordListRequests() {
  const requests: URLSearchParams[] = [];
  server.use(
    http.get('/api/v1/subscriptions', ({ request }) => {
      requests.push(new URL(request.url).searchParams);
      return HttpResponse.json({
        items: [subscriptionWire(SIGUR, 'Sigur Rós')],
        total: PAGE_SIZE * 2 + 1,
        facets: { countries: ['IS'], types: ['Group'] },
      });
    }),
  );
  return requests;
}

describe('useSubscriptionsController', () => {
  it('builds cards from subscriptions', async () => {
    server.use(
      http.get('/api/v1/subscriptions', () =>
        HttpResponse.json({
          items: [
            subscriptionWire(NIRVANA, 'Nirvana', '2026-10-17T06:03:00Z'),
            subscriptionWire(SIGUR, 'Sigur Rós'),
          ],
          total: 2,
        }),
      ),
    );
    const { result } = renderController();

    await waitFor(() => expect(result.current.cards).toHaveLength(2));
    expect(result.current.total).toBe(2);
    expect(result.current.pageCount).toBe(1);
    expect(result.current.cards[0]).toMatchObject({
      name: 'Nirvana',
      synced: true,
      typesText: 'Альбом, EP',
      typesSource: 'own',
      featuring: false,
      featuringSource: 'global',
    });
    expect(result.current.cards[1]).toMatchObject({
      synced: false,
      syncText: 'Ждёт первой проверки',
    });
  });

  it('reports an empty drawer only when nothing is subscribed', async () => {
    server.use(http.get('/api/v1/subscriptions', () => HttpResponse.json({ items: [], total: 0 })));
    const { result } = renderController();
    await waitFor(() => expect(result.current.isEmpty).toBe(true));

    act(() => result.current.onSearchChange('abc'));
    await waitFor(() => expect(result.current.isEmpty).toBe(false));
  });

  it('pages through the list and resets the page on a new filter', async () => {
    const offsets: string[] = [];
    server.use(
      http.get('/api/v1/subscriptions', ({ request }) => {
        offsets.push(new URL(request.url).searchParams.get('offset') ?? '');
        return HttpResponse.json({
          items: [subscriptionWire(SIGUR, 'Sigur Rós')],
          total: PAGE_SIZE * 2 + 1,
        });
      }),
    );
    const { result } = renderController();
    await waitFor(() => expect(result.current.pageCount).toBe(3));

    act(() => result.current.onPageChange(1));
    await waitFor(() => expect(offsets).toContain(String(PAGE_SIZE)));
    expect(result.current.page).toBe(1);
    expect(result.current.range).toMatchObject({ from: PAGE_SIZE + 1 });

    act(() => result.current.onSearchChange('sig'));
    expect(result.current.page).toBe(0);
  });

  it('reads the page and filters from the URL and sends them to the API', async () => {
    const requests = recordListRequests();
    const { result } = renderController('/artists?page=2&size=24&status=pending&country=IS');

    await waitFor(() => expect(result.current.cards).toHaveLength(1));
    const q = requests.at(-1)!;
    expect(q.get('offset')).toBe('24');
    expect(q.get('limit')).toBe('24');
    expect(q.get('status')).toBe('pending');
    expect(q.get('country')).toBe('IS');
    expect(result.current.filters).toMatchObject({
      status: 'pending',
      country: 'IS',
      sort: 'name',
    });
    expect(result.current.facets.countries).toEqual(['IS']);
    expect(result.current.filtered).toBe(true);
  });

  it('starts from the first page on a new filter and resets everything at once', async () => {
    const requests = recordListRequests();
    const { result } = renderController('/artists?page=3&q=sig');
    await waitFor(() => expect(result.current.page).toBe(2));

    act(() => result.current.onFilterChange('type', 'Group'));
    await waitFor(() => expect(requests.at(-1)?.get('type')).toBe('Group'));
    expect(result.current.page).toBe(0);

    act(() => result.current.onResetFilters());
    await waitFor(() => expect(result.current.filtered).toBe(false));
    expect(result.current.search).toBe('');
  });

  it('moves back to the last page when the requested one is past the end', async () => {
    recordListRequests(); // 3 pages
    const { result } = renderController('/artists?page=9');
    await waitFor(() => expect(result.current.page).toBe(2));
  });

  it('unsubscribes and refetches the list', async () => {
    let subscribed = true;
    server.use(
      http.get('/api/v1/subscriptions', () =>
        HttpResponse.json(
          subscribed
            ? { items: [subscriptionWire(SIGUR, 'Sigur Rós')], total: 1 }
            : { items: [], total: 0 },
        ),
      ),
      http.delete(`/api/v1/subscriptions/${SIGUR}`, () => {
        subscribed = false;
        return new HttpResponse(null, { status: 204 });
      }),
    );
    const { result } = renderController();
    await waitFor(() => expect(result.current.cards).toHaveLength(1));

    act(() => result.current.onUnsubscribe(SIGUR));
    await waitFor(() => expect(result.current.cards).toHaveLength(0));
    expect(result.current.isEmpty).toBe(true);
  });
});
