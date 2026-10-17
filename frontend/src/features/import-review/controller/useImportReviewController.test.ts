import { act, renderHook, waitFor } from '@testing-library/react';
import { http, HttpResponse } from 'msw';
import { importItemWire, importJobWire } from '@/test/fixtures';
import { createWrapper } from '@/test/queryWrapper';
import { server } from '@/test/server';
import { OTHER, useImportReviewController } from './useImportReviewController';

const UK = '9282c8b4-ca0b-4c6b-b7e3-4f7762dfc4d6';

function renderController(id = 'job-1') {
  const { Wrapper, queryClient } = createWrapper();
  const hook = renderHook(() => useImportReviewController(id), { wrapper: Wrapper });
  return { ...hook, queryClient };
}

describe('useImportReviewController', () => {
  it('lists the open lines as cards and stamps a chosen candidate', async () => {
    let open = [
      importItemWire('item-1', 'Nirvana'),
      importItemWire('item-2', 'Zzqxj', { status: 'not_found', candidates: [] }),
    ];
    let status: string | null = null;
    let body: unknown;
    server.use(
      http.get('/api/v1/imports/job-1', () =>
        HttpResponse.json(importJobWire({ matched: 3, needs_review: 1, not_found: 1 })),
      ),
      http.get('/api/v1/imports/job-1/items', ({ request }) => {
        status = new URL(request.url).searchParams.get('status');
        return HttpResponse.json({ items: open });
      }),
      http.post('/api/v1/imports/job-1/items/item-1/resolve', async ({ request }) => {
        body = await request.json();
        open = open.slice(1);
        return HttpResponse.json(importItemWire('item-1', 'Nirvana', { status: 'matched' }));
      }),
    );
    const { result, queryClient } = renderController();
    const invalidate = vi.spyOn(queryClient, 'invalidateQueries');

    await waitFor(() => expect(result.current.cards).toHaveLength(2));
    expect(status).toBe('needs_review,not_found,failed');
    expect(result.current.counts).toMatchObject({ total: 5, matched: 3, toReview: 2 });
    expect(result.current.processing).toBe(false);

    const [nirvana, unknown] = result.current.cards;
    expect(nirvana).toMatchObject({
      rawValue: 'Nirvana',
      statusText: 'несколько тёзок',
      choice: undefined,
      canConfirm: false,
      featured: false,
    });
    expect(nirvana?.candidates[1]).toEqual({
      mbid: UK,
      name: 'Nirvana',
      details: '60s band from the UK · Группа · GB · 1965–',
      score: 100,
    });
    // Nothing found: the manual MBID field is the only choice.
    expect(unknown).toMatchObject({ choice: OTHER, canConfirm: true });

    act(() => result.current.onChoose('item-1', UK));
    act(() => result.current.onFeaturedChange('item-1', true));
    expect(result.current.cards[0]).toMatchObject({ choice: UK, featured: true, canConfirm: true });
    act(() => result.current.onConfirm('item-1'));

    await waitFor(() => expect(result.current.cards).toHaveLength(1));
    expect(body).toEqual({ mbid: UK, include_featured: true });
    expect(invalidate).toHaveBeenCalledWith({ queryKey: ['subscriptions'] });
  });

  it('keeps the CSV featuring value unless changed and checks the manual MBID', async () => {
    let body: unknown;
    server.use(
      http.get('/api/v1/imports/job-1', () => HttpResponse.json(importJobWire({ failed: 1 }))),
      http.get('/api/v1/imports/job-1/items', () =>
        HttpResponse.json({
          items: [
            importItemWire('item-1', 'Radiohead', {
              status: 'failed',
              candidates: [],
              include_featured: true,
              error: 'upstream unavailable',
            }),
          ],
        }),
      ),
      http.post('/api/v1/imports/job-1/items/item-1/resolve', async ({ request }) => {
        body = await request.json();
        return HttpResponse.json(
          {
            title: 'Validation failed',
            errors: [{ field: 'mbid', message: 'MusicBrainz does not know this artist' }],
          },
          { status: 422 },
        );
      }),
    );
    const { result } = renderController();
    await waitFor(() => expect(result.current.cards).toHaveLength(1));
    expect(result.current.cards[0]).toMatchObject({
      featured: true,
      error: 'upstream unavailable',
      statusText: 'ошибка',
    });

    act(() => result.current.onConfirm('item-1'));
    expect(result.current.cards[0]?.otherError).toMatch(/MBID/);
    expect(body).toBeUndefined();

    act(() => result.current.onOtherChange('item-1', ' https://musicbrainz.org/artist/x '));
    act(() => result.current.onConfirm('item-1'));
    await waitFor(() =>
      expect(result.current.cards[0]?.errorText).toBe(
        'Не принято: MusicBrainz does not know this artist',
      ),
    );
    expect(body).toEqual({ mbid: 'https://musicbrainz.org/artist/x' });
  });

  it('skips a line and shows the other tabs as plain rows', async () => {
    let body: unknown;
    let status: string | null = null;
    server.use(
      http.get('/api/v1/imports/job-1', () =>
        HttpResponse.json(importJobWire({ matched: 1, needs_review: 1 })),
      ),
      http.get('/api/v1/imports/job-1/items', ({ request }) => {
        status = new URL(request.url).searchParams.get('status');
        return HttpResponse.json({
          items:
            status === 'matched'
              ? [
                  importItemWire('item-9', 'Björk', {
                    status: 'matched',
                    artist: { mbid: 'b-1', name: 'Björk' },
                  }),
                ]
              : [importItemWire('item-1', 'Nirvana')],
        });
      }),
      http.post('/api/v1/imports/job-1/items/item-1/resolve', async ({ request }) => {
        body = await request.json();
        return HttpResponse.json(importItemWire('item-1', 'Nirvana', { status: 'skipped' }));
      }),
    );
    const { result } = renderController();
    await waitFor(() => expect(result.current.cards).toHaveLength(1));

    act(() => result.current.onSkip('item-1'));
    await waitFor(() => expect(body).toEqual({ skip: true }));

    act(() => result.current.onTabChange('matched'));
    await waitFor(() => expect(result.current.rows).toHaveLength(1));
    expect(status).toBe('matched');
    expect(result.current.cards).toEqual([]);
    expect(result.current.rows[0]).toMatchObject({ rawValue: 'Björk', artist: { name: 'Björk' } });
  });

  it('reports an unknown import', async () => {
    server.use(
      http.get('/api/v1/imports/nope', () =>
        HttpResponse.json({ title: 'Not found' }, { status: 404 }),
      ),
      http.get('/api/v1/imports/nope/items', () =>
        HttpResponse.json({ title: 'Not found' }, { status: 404 }),
      ),
    );
    const { result } = renderController('nope');
    await waitFor(() => expect(result.current.notFound).toBe(true));
  });
});
