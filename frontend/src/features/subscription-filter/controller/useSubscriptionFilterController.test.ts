import { act, renderHook, waitFor } from '@testing-library/react';
import { http, HttpResponse } from 'msw';
import { subscriptionWire } from '@/test/fixtures';
import { createWrapper } from '@/test/queryWrapper';
import { server } from '@/test/server';
import { useSubscriptionFilterController } from './useSubscriptionFilterController';

const SIGUR = 'f6f2326f-6b25-4170-b89d-e235b25508e8';

/** The server applies PATCH bodies to a filter kept here, like the real backend. */
function serveSubscription() {
  const sub = subscriptionWire(SIGUR, 'Sigur Rós');
  const globals = { primary_types: ['album', 'ep', 'single'], include_featured: false };
  const patches: unknown[] = [];
  server.use(
    http.get(`/api/v1/subscriptions/${SIGUR}`, () => HttpResponse.json(sub)),
    http.patch(`/api/v1/subscriptions/${SIGUR}`, async ({ request }) => {
      const body = (await request.json()) as Record<string, unknown>;
      patches.push(body);
      if ('primary_types' in body) {
        sub.filter.primary_types =
          body.primary_types === null
            ? { value: globals.primary_types, overridden: false }
            : { value: body.primary_types as string[], overridden: true };
      }
      if ('include_featured' in body) {
        sub.filter.include_featured =
          body.include_featured === null
            ? { value: globals.include_featured, overridden: false }
            : { value: body.include_featured as boolean, overridden: true };
      }
      return HttpResponse.json(sub);
    }),
  );
  return patches;
}

function renderController() {
  const { Wrapper } = createWrapper();
  return renderHook(() => useSubscriptionFilterController(SIGUR), { wrapper: Wrapper });
}

describe('useSubscriptionFilterController', () => {
  it('takes a field over, edits it and gives it back to the global settings', async () => {
    const patches = serveSubscription();
    const { result } = renderController();
    await waitFor(() => expect(result.current.ready).toBe(true));
    // The fixture has an own primary types override and global featuring.
    expect(result.current.ownTypes).toBe(true);
    expect(result.current.ownFeatured).toBe(false);

    act(() => result.current.onOwnFeaturedChange(true));
    await waitFor(() => expect(result.current.ownFeatured).toBe(true));
    act(() => result.current.onIncludeFeaturedChange(true));
    await waitFor(() => expect(result.current.includeFeatured).toBe(true));

    act(() => result.current.onTogglePrimaryType('ep'));
    await waitFor(() => expect(result.current.primaryTypes).toEqual(['album']));

    act(() => result.current.onOwnTypesChange(false));
    await waitFor(() => expect(result.current.ownTypes).toBe(false));
    expect(result.current.primaryTypes).toEqual(['album', 'ep', 'single']);

    expect(patches).toEqual([
      { include_featured: false },
      { include_featured: true },
      { primary_types: ['album'] },
      { primary_types: null },
    ]);
  });
});
