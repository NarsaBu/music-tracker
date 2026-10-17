import type { z } from 'zod';
import { ApiContractError, ApiError, type ApiFieldError } from './ApiError';

/**
 * Transport abstraction the Model layer depends on (DIP): repositories receive
 * an HttpClient and never call fetch directly, so tests can substitute it.
 * Every response is validated against a zod schema at the boundary.
 */
export interface HttpClient {
  get<S extends z.ZodType>(path: string, schema: S): Promise<z.infer<S>>;
  post<S extends z.ZodType>(path: string, body: unknown, schema: S): Promise<z.infer<S>>;
  put<S extends z.ZodType>(path: string, body: unknown, schema: S): Promise<z.infer<S>>;
  patch<S extends z.ZodType>(path: string, body: unknown, schema: S): Promise<z.infer<S>>;
  delete(path: string): Promise<void>;
}

export interface FetchHttpClientOptions {
  baseUrl?: string;
  fetchImpl?: typeof fetch;
  /** Called on any 401, e.g. to redirect to the login screen. */
  onUnauthorized?: () => void;
}

interface ProblemBody {
  title?: string;
  detail?: string;
  errors?: ApiFieldError[];
}

export class FetchHttpClient implements HttpClient {
  private readonly baseUrl: string;
  private readonly fetchImpl: typeof fetch;
  private readonly onUnauthorized?: () => void;

  constructor({ baseUrl = '', fetchImpl, onUnauthorized }: FetchHttpClientOptions = {}) {
    this.baseUrl = baseUrl;
    // Bind lazily so tests can stub globalThis.fetch after construction.
    this.fetchImpl = fetchImpl ?? ((...args) => globalThis.fetch(...args));
    this.onUnauthorized = onUnauthorized;
  }

  get<S extends z.ZodType>(path: string, schema: S) {
    return this.request(path, { method: 'GET' }, schema);
  }

  post<S extends z.ZodType>(path: string, body: unknown, schema: S) {
    return this.request(path, { method: 'POST', body: JSON.stringify(body) }, schema);
  }

  put<S extends z.ZodType>(path: string, body: unknown, schema: S) {
    return this.request(path, { method: 'PUT', body: JSON.stringify(body) }, schema);
  }

  patch<S extends z.ZodType>(path: string, body: unknown, schema: S) {
    return this.request(path, { method: 'PATCH', body: JSON.stringify(body) }, schema);
  }

  async delete(path: string): Promise<void> {
    await this.send(path, { method: 'DELETE' });
  }

  private async request<S extends z.ZodType>(
    path: string,
    init: RequestInit,
    schema: S,
  ): Promise<z.infer<S>> {
    const res = await this.send(path, init);
    const data: unknown = res.status === 204 ? undefined : await res.json();
    const parsed = schema.safeParse(data);
    if (!parsed.success) throw new ApiContractError(path, parsed.error);
    return parsed.data;
  }

  private async send(path: string, init: RequestInit): Promise<Response> {
    const headers = new Headers(init.headers);
    headers.set('Accept', 'application/json');
    if (init.body !== undefined) headers.set('Content-Type', 'application/json');

    const res = await this.fetchImpl(this.baseUrl + path, {
      ...init,
      headers,
      credentials: 'same-origin',
    });
    if (res.ok) return res;

    const problem = await readProblem(res);
    const error = new ApiError(
      res.status,
      problem.title ?? res.statusText,
      problem.detail,
      problem.errors,
    );
    if (error.isUnauthorized) this.onUnauthorized?.();
    throw error;
  }
}

async function readProblem(res: Response): Promise<ProblemBody> {
  try {
    return (await res.json()) as ProblemBody;
  } catch {
    return {};
  }
}
