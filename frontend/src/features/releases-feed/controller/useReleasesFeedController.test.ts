import { act, renderHook, waitFor } from '@testing-library/react';
import { http, HttpResponse } from 'msw';
import { feedReleaseWire } from '@/test/fixtures';
import { createWrapper } from '@/test/queryWrapper';
import { server } from '@/test/server';
import { useReleasesFeedController } from './useReleasesFeedController';

function renderController() {
  const { Wrapper } = createWrapper();
  return renderHook(() => useReleasesFeedController(), { wrapper: Wrapper });
}

describe('useReleasesFeedController', () => {
  it('pages through the feed with the cursor', async () => {
    const cursors: Array<string | null> = [];
    server.use(
      http.get('/api/v1/releases', ({ request }) => {
        const cursor = new URL(request.url).searchParams.get('cursor');
        cursors.push(cursor);
        return HttpResponse.json(
          cursor
            ? { items: [feedReleaseWire(2)], next_cursor: null }
            : {
                items: [
                  feedReleaseWire(1, {
                    is_announcement: true,
                    found_via: { mbid: 'x', name: 'Björk', credit_role: 'featured' },
                  }),
                ],
                next_cursor: 'c1',
              },
        );
      }),
    );
    const { result } = renderController();

    await waitFor(() => expect(result.current.records).toHaveLength(1));
    expect(result.current.hasMore).toBe(true);
    expect(result.current.lcdText).toBe('001+');
    expect(result.current.records[0]).toMatchObject({
      featured: true,
      announcement: true,
      typeLabel: 'Альбом',
      date: '10 окт 2026',
    });
    expect(result.current.records[0]?.foundText).toMatch(/^Гость: Björk/);

    act(() => result.current.onLoadMore());
    await waitFor(() => expect(result.current.records).toHaveLength(2));
    expect(result.current.hasMore).toBe(false);
    expect(cursors).toEqual([null, 'c1']);
  });

  it('sends filters and tells an empty filter from an empty feed', async () => {
    const queries: string[] = [];
    server.use(
      http.get('/api/v1/releases', ({ request }) => {
        queries.push(new URL(request.url).search);
        return HttpResponse.json({ items: [], next_cursor: null });
      }),
    );
    const { result } = renderController();
    await waitFor(() => expect(result.current.isEmpty).toBe(true));
    expect(result.current.filtered).toBe(false);

    act(() => result.current.onTypeChange('single'));
    act(() => result.current.onRoleChange('featured'));
    await waitFor(() => expect(queries.at(-1)).toContain('role=featured'));
    expect(queries.at(-1)).toContain('type=single');
    await waitFor(() => expect(result.current.isEmpty).toBe(true));
    expect(result.current.filtered).toBe(true);
    expect(queries.at(-1)).not.toContain('sort=');

    act(() => result.current.onSortChange('released'));
    await waitFor(() => expect(queries.at(-1)).toContain('sort=released'));
  });
});
