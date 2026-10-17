import { z } from 'zod';
import { ApiContractError, ApiError } from './ApiError';
import { FetchHttpClient } from './HttpClient';

const schema = z.object({ status: z.string() });

function jsonResponse(status: number, body: unknown, contentType = 'application/json') {
  return new Response(JSON.stringify(body), { status, headers: { 'Content-Type': contentType } });
}

describe('FetchHttpClient', () => {
  it('returns parsed data on success', async () => {
    const fetchImpl = vi.fn().mockResolvedValue(jsonResponse(200, { status: 'ok' }));
    const client = new FetchHttpClient({ baseUrl: '/base', fetchImpl });

    await expect(client.get('/healthz', schema)).resolves.toEqual({ status: 'ok' });
    expect(fetchImpl).toHaveBeenCalledWith(
      '/base/healthz',
      expect.objectContaining({ method: 'GET' }),
    );
  });

  it('sends JSON body with content type', async () => {
    const fetchImpl = vi.fn().mockResolvedValue(jsonResponse(200, { status: 'ok' }));
    const client = new FetchHttpClient({ fetchImpl });

    await client.post('/x', { a: 1 }, schema);

    const init = fetchImpl.mock.calls[0]![1] as RequestInit;
    expect(init.body).toBe('{"a":1}');
    expect(new Headers(init.headers).get('Content-Type')).toBe('application/json');
  });

  it('leaves the multipart content type of a form upload to fetch', async () => {
    const fetchImpl = vi.fn().mockResolvedValue(jsonResponse(202, { status: 'ok' }));
    const client = new FetchHttpClient({ fetchImpl });
    const form = new FormData();
    form.append('file', new Blob(['Radiohead']), 'a.txt');

    await client.postForm('/x', form, schema);

    const init = fetchImpl.mock.calls[0]![1] as RequestInit;
    expect(init.body).toBe(form);
    expect(new Headers(init.headers).has('Content-Type')).toBe(false);
  });

  it('treats an empty body as undefined, whatever the status', async () => {
    const fetchImpl = vi.fn().mockResolvedValue(new Response(null, { status: 202 }));
    const client = new FetchHttpClient({ fetchImpl });

    await expect(client.post('/x', undefined, z.undefined())).resolves.toBeUndefined();
  });

  it('maps problem details to ApiError', async () => {
    const fetchImpl = vi.fn().mockResolvedValue(
      jsonResponse(
        422,
        {
          title: 'Validation failed',
          detail: 'bad mbid',
          errors: [{ field: 'mbid', message: 'invalid' }],
        },
        'application/problem+json',
      ),
    );
    const client = new FetchHttpClient({ fetchImpl });

    const err = await client.get('/x', schema).catch((e: unknown) => e);
    expect(err).toBeInstanceOf(ApiError);
    expect(err).toMatchObject({ status: 422, title: 'Validation failed', detail: 'bad mbid' });
    expect((err as ApiError).fields).toEqual([{ field: 'mbid', message: 'invalid' }]);
  });

  it('rejects responses that violate the schema', async () => {
    const fetchImpl = vi.fn().mockResolvedValue(jsonResponse(200, { status: 42 }));
    const client = new FetchHttpClient({ fetchImpl });

    await expect(client.get('/x', schema)).rejects.toBeInstanceOf(ApiContractError);
  });
});
