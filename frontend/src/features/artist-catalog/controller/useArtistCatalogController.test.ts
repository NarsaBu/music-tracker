import { act, renderHook, waitFor } from '@testing-library/react';
import { http, HttpResponse } from 'msw';
import { releaseGroupWire, subscriptionWire } from '@/test/fixtures';
import { createWrapper } from '@/test/queryWrapper';
import { server } from '@/test/server';
import { useArtistCatalogController } from './useArtistCatalogController';

const SIGUR = 'f6f2326f-6b25-4170-b89d-e235b25508e8';
const RG1 = '00000000-0000-0000-0000-000000000001';
const RG2 = '00000000-0000-0000-0000-000000000002';

function renderController(mbid = SIGUR) {
  const { Wrapper } = createWrapper();
  return renderHook(() => useArtistCatalogController(mbid), { wrapper: Wrapper });
}

describe('useArtistCatalogController', () => {
  it('maps the catalog and filters by role on the server', async () => {
    const roles: Array<string | null> = [];
    server.use(
      http.get(`/api/v1/subscriptions/${SIGUR}`, () =>
        HttpResponse.json(subscriptionWire(SIGUR, 'Sigur Rós', '2026-10-17T06:03:00Z')),
      ),
      http.get(`/api/v1/artists/${SIGUR}/release-groups`, ({ request }) => {
        const role = new URL(request.url).searchParams.get('role');
        roles.push(role);
        return HttpResponse.json({
          items:
            role === 'featured'
              ? [releaseGroupWire(RG2, 'Guest spot', 'featured')]
              : [
                  releaseGroupWire(RG1, 'Ágætis byrjun'),
                  releaseGroupWire(RG2, 'Guest spot', 'featured', {
                    is_baseline: false,
                    is_announcement: true,
                    first_release_date: '2027-02',
                    primary_type: null,
                  }),
                ],
        });
      }),
    );
    const { result } = renderController();

    await waitFor(() => expect(result.current.records).toHaveLength(2));
    expect(result.current.name).toBe('Sigur Rós');
    expect(result.current.synced).toBe(true);
    expect(result.current.records[0]).toMatchObject({
      date: 'июн 1999',
      typeLabel: 'Album',
      fresh: false,
    });
    expect(result.current.records[1]).toMatchObject({
      featured: true,
      announcement: true,
      fresh: true,
      typeLabel: null,
      date: 'фев 2027',
    });

    act(() => result.current.onRoleChange('featured'));
    await waitFor(() => expect(result.current.records).toHaveLength(1));
    expect(roles).toEqual([null, 'featured']);
  });

  it('"check now" polls until the sync time changes, then reloads the catalog', async () => {
    let synced: string | null = null;
    let catalogCalls = 0;
    server.use(
      http.get(`/api/v1/subscriptions/${SIGUR}`, () =>
        HttpResponse.json(subscriptionWire(SIGUR, 'Sigur Rós', synced)),
      ),
      http.get(`/api/v1/artists/${SIGUR}/release-groups`, () => {
        catalogCalls++;
        return HttpResponse.json({ items: synced ? [releaseGroupWire(RG1, 'Von')] : [] });
      }),
      http.post(`/api/v1/subscriptions/${SIGUR}/sync`, () => {
        synced = '2026-10-17T07:00:00Z';
        return new HttpResponse(null, { status: 202 });
      }),
    );
    const { result } = renderController();
    await waitFor(() => expect(result.current.isLoading).toBe(false));
    expect(result.current.synced).toBe(false);

    act(() => result.current.onCheckNow());
    expect(result.current.checking).toBe(true);

    await waitFor(() => expect(result.current.checking).toBe(false), { timeout: 4000 });
    expect(result.current.synced).toBe(true);
    await waitFor(() => expect(result.current.records).toHaveLength(1));
    expect(catalogCalls).toBeGreaterThanOrEqual(2);
  });

  it('reports a missing subscription', async () => {
    server.use(
      http.get(`/api/v1/subscriptions/${SIGUR}`, () =>
        HttpResponse.json({ title: 'Not found' }, { status: 404 }),
      ),
      http.get(`/api/v1/artists/${SIGUR}/release-groups`, () =>
        HttpResponse.json({ title: 'Not found' }, { status: 404 }),
      ),
    );
    const { result } = renderController();
    await waitFor(() => expect(result.current.notSubscribed).toBe(true));
  });
});
