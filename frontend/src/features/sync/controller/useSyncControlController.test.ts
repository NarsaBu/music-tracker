import { act, renderHook, waitFor } from '@testing-library/react';
import { http, HttpResponse } from 'msw';
import { syncRunWire } from '@/test/fixtures';
import { createWrapper } from '@/test/queryWrapper';
import { server } from '@/test/server';
import { useSyncControlController } from './useSyncControlController';

function renderController() {
  const { Wrapper, queryClient } = createWrapper();
  const hook = renderHook(() => useSyncControlController(), { wrapper: Wrapper });
  return { ...hook, queryClient };
}

describe('useSyncControlController', () => {
  it('shows that nothing has run yet', async () => {
    server.use(http.get('/api/v1/sync/runs/latest', () => HttpResponse.json(null)));
    const { result } = renderController();
    await waitFor(() => expect(result.current.lcdText).toBe('НЕТ ПРОГОНОВ'));
    expect(result.current.running).toBe(false);
    expect(result.current.progress).toBe(0);
    expect(result.current.counterText).toBe('---/---');
  });

  it('starts a run, follows its progress and refreshes data when it ends', async () => {
    let latest = syncRunWire(0, 4, false);
    server.use(
      http.get('/api/v1/sync/runs/latest', () => HttpResponse.json(latest)),
      http.post('/api/v1/sync', () => HttpResponse.json(latest, { status: 202 })),
    );
    const { result, queryClient } = renderController();
    const invalidate = vi.spyOn(queryClient, 'invalidateQueries');

    await waitFor(() => expect(result.current.running).toBe(true));
    act(() => result.current.onStart());
    expect(result.current.lcdText).toBe('000%');

    latest = syncRunWire(2, 4, false);
    await waitFor(() => expect(result.current.progress).toBe(50), { timeout: 4000 });
    expect(result.current.lcdText).toBe('050%');
    expect(result.current.counterText).toBe('002/004');
    expect(result.current.progressText).toBe('2 из 4');

    latest = syncRunWire(4, 4, true, 3);
    await waitFor(() => expect(result.current.running).toBe(false), { timeout: 4000 });
    expect(result.current.lcdText).toBe('НОВЫХ 003');
    expect(result.current.progress).toBe(100);
    expect(result.current.statusText).toMatch(/новинок: 3/);
    expect(invalidate).toHaveBeenCalledWith({ queryKey: ['subscriptions'] });
  });
});
