import { act, renderHook, waitFor } from '@testing-library/react';
import { http, HttpResponse } from 'msw';
import { importJobWire } from '@/test/fixtures';
import { createWrapper } from '@/test/queryWrapper';
import { server } from '@/test/server';
import { MAX_FILE_BYTES, useImportsController } from './useImportsController';

function renderController() {
  const { Wrapper } = createWrapper();
  return renderHook(() => useImportsController(), { wrapper: Wrapper });
}

describe('useImportsController', () => {
  it('asks for a tape when nothing was imported', async () => {
    server.use(http.get('/api/v1/imports', () => HttpResponse.json({ items: [] })));
    const { result } = renderController();
    await waitFor(() => expect(result.current.isLoading).toBe(false));
    expect(result.current.lcdText).toBe('ВСТАВЬТЕ КАССЕТУ');
    expect(result.current.rows).toEqual([]);
    expect(result.current.spinning).toBe(false);
  });

  it('uploads a file and follows the import until it is done', async () => {
    let jobs: ReturnType<typeof importJobWire>[] = [];
    let uploads = 0;
    server.use(
      http.get('/api/v1/imports', () => HttpResponse.json({ items: jobs })),
      // The body is not read: jsdom's FormData does not survive Node's fetch.
      http.post('/api/v1/imports', () => {
        uploads++;
        jobs = [importJobWire({ pending: 3, matched: 1 })];
        return HttpResponse.json(jobs[0], { status: 202 });
      }),
    );
    const { result } = renderController();
    await waitFor(() => expect(result.current.isLoading).toBe(false));

    act(() => result.current.onFile(new File(['Radiohead'], 'artists.txt')));
    expect(result.current.tapeLabel).toBe('artists.txt');

    await waitFor(() => expect(result.current.rows).toHaveLength(1));
    expect(uploads).toBe(1);
    expect(result.current.spinning).toBe(true);
    expect(result.current.lcdText).toBe('▶ 001/004');
    expect(result.current.rows[0]).toMatchObject({
      progress: 25,
      led: 'on',
      processing: true,
      statusText: 'Идёт сопоставление · добавлено 1',
    });

    jobs = [importJobWire({ matched: 2, needs_review: 1, not_found: 1 })];
    await waitFor(() => expect(result.current.rows[0]?.processing).toBe(false), { timeout: 4000 });
    expect(result.current.lcdText).toBe('СТОП');
    expect(result.current.rows[0]).toMatchObject({
      counterText: '004/004',
      led: 'warn',
      toReview: 2,
      statusText: 'добавлено 2 · ждут разбора 2',
    });
  });

  it('rejects a big file without sending it and shows server validation errors', async () => {
    let uploads = 0;
    server.use(
      http.get('/api/v1/imports', () => HttpResponse.json({ items: [] })),
      http.post('/api/v1/imports', () => {
        uploads++;
        return HttpResponse.json(
          {
            title: 'Validation failed',
            errors: [
              { field: 'file', message: 'line 3: include_featured: "maybe" is not true/false' },
            ],
          },
          { status: 422 },
        );
      }),
    );
    const { result } = renderController();

    const big = new File(['x'], 'big.csv');
    Object.defineProperty(big, 'size', { value: MAX_FILE_BYTES + 1 });
    act(() => result.current.onFile(big));
    expect(result.current.uploadError).toMatch(/больше 1 МБ/);

    act(() => result.current.onFile(new File(['x'], 'bad.csv')));
    await waitFor(() =>
      expect(result.current.uploadError).toBe(
        'Файл не принят: line 3: include_featured: "maybe" is not true/false',
      ),
    );
    expect(uploads).toBe(1);
  });
});
