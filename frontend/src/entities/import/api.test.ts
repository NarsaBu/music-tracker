import { http, HttpResponse } from 'msw';
import { httpClient, type HttpClient } from '@/shared/api';
import { importItemWire, importJobWire } from '@/test/fixtures';
import { server } from '@/test/server';
import { createImportApi } from './api';
import { progressOf, toReviewOf } from './model';

const api = createImportApi(httpClient);

describe('import api', () => {
  it('uploads the file as multipart form data', async () => {
    // jsdom's FormData does not survive Node's fetch, so the transport is substituted (DIP).
    const postForm = vi.fn().mockResolvedValue({
      ...importJobWire({ pending: 2 }),
      createdAt: '2026-10-17T06:00:00Z',
      finishedAt: null,
    });
    const fakeHttp: HttpClient = {
      get: vi.fn(),
      post: vi.fn(),
      put: vi.fn(),
      patch: vi.fn(),
      delete: vi.fn(),
      postForm,
    };
    const file = new File(['Radiohead\nBjörk\n'], 'artists.txt');

    const job = await createImportApi(fakeHttp).upload(file);

    const [path, form] = postForm.mock.calls[0] as [string, FormData];
    expect(path).toBe('/api/v1/imports');
    expect(form.get('file')).toBe(file);
    expect(progressOf(job)).toBe(0);
  });

  it('maps items and sends the resolution', async () => {
    let status: string | null = null;
    let body: unknown;
    server.use(
      http.get('/api/v1/imports/job-1/items', ({ request }) => {
        status = new URL(request.url).searchParams.get('status');
        return HttpResponse.json({ items: [importItemWire('item-1', 'Nirvana')] });
      }),
      http.post('/api/v1/imports/job-1/items/item-1/resolve', async ({ request }) => {
        body = await request.json();
        return HttpResponse.json(importItemWire('item-1', 'Nirvana', { status: 'matched' }));
      }),
    );

    const items = await api.items('job-1', ['needs_review', 'not_found']);
    expect(status).toBe('needs_review,not_found');
    expect(items[0]).toMatchObject({
      lineNo: 2,
      rawValue: 'Nirvana',
      includeFeatured: null,
    });
    expect(items[0]?.candidates[1]).toMatchObject({
      sortName: 'Nirvana',
      country: 'GB',
      lifeSpan: { begin: '1965', end: null },
    });

    await api.resolve('job-1', 'item-1', { mbid: 'm-1', includeFeatured: true });
    expect(body).toEqual({ mbid: 'm-1', include_featured: true });
    await api.resolve('job-1', 'item-1', { skip: true });
    expect(body).toEqual({ skip: true });
  });

  it('computes progress and the review backlog', () => {
    const job = {
      ...importJobWire({ pending: 1, matched: 5, needs_review: 2, not_found: 1, failed: 1 }),
      createdAt: '',
      finishedAt: null,
    };
    expect(progressOf(job)).toBe(90);
    expect(toReviewOf(job)).toBe(4);
  });
});
