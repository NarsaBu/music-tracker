import { z } from 'zod';
import type { HttpClient } from '@/shared/api';
import {
  IMPORT_ITEM_STATUSES,
  type ImportItem,
  type ImportItemStatus,
  type ImportJob,
  type ResolveRequest,
} from './model';

const countsSchema = z.object(
  Object.fromEntries(IMPORT_ITEM_STATUSES.map((s) => [s, z.number()])) as Record<
    ImportItemStatus,
    z.ZodNumber
  >,
);

const jobSchema = z
  .object({
    id: z.string(),
    filename: z.string(),
    total: z.number(),
    counts: countsSchema,
    finished: z.boolean(),
    created_at: z.string(),
    finished_at: z.string().nullable(),
  })
  .transform(({ created_at, finished_at, ...rest }): ImportJob => ({
    ...rest,
    createdAt: created_at,
    finishedAt: finished_at,
  }));

const candidateSchema = z
  .object({
    mbid: z.string(),
    name: z.string(),
    sort_name: z.string(),
    disambiguation: z.string(),
    type: z.string().nullable(),
    country: z.string().nullable(),
    life_span: z.object({
      begin: z.string().nullable(),
      end: z.string().nullable(),
      ended: z.boolean(),
    }),
    score: z.number(),
  })
  .transform(({ sort_name, life_span, ...rest }) => ({
    ...rest,
    sortName: sort_name,
    lifeSpan: life_span,
  }));

const itemSchema = z
  .object({
    id: z.string(),
    line_no: z.number(),
    raw_value: z.string(),
    input_mbid: z.string().nullable(),
    include_featured: z.boolean().nullable(),
    status: z.enum(IMPORT_ITEM_STATUSES),
    candidates: z.array(candidateSchema),
    artist: z.object({ mbid: z.string(), name: z.string() }).nullable(),
    error: z.string().nullable(),
  })
  .transform(({ line_no, raw_value, input_mbid, include_featured, ...rest }): ImportItem => ({
    ...rest,
    lineNo: line_no,
    rawValue: raw_value,
    inputMbid: input_mbid,
    includeFeatured: include_featured,
  }));

const jobsSchema = z.object({ items: z.array(jobSchema) });
const itemsSchema = z.object({ items: z.array(itemSchema) });

export function createImportApi(http: HttpClient) {
  return {
    /** Uploads a .txt/.csv file; lines are matched in the background (202). */
    upload: (file: File): Promise<ImportJob> => {
      const form = new FormData();
      form.append('file', file);
      return http.postForm('/api/v1/imports', form, jobSchema);
    },
    list: async (): Promise<ImportJob[]> => (await http.get('/api/v1/imports', jobsSchema)).items,
    get: (id: string): Promise<ImportJob> =>
      http.get(`/api/v1/imports/${encodeURIComponent(id)}`, jobSchema),
    items: async (
      id: string,
      statuses: readonly ImportItemStatus[] = [],
    ): Promise<ImportItem[]> => {
      const query = statuses.length
        ? `?${new URLSearchParams({ status: statuses.join(',') })}`
        : '';
      const res = await http.get(
        `/api/v1/imports/${encodeURIComponent(id)}/items${query}`,
        itemsSchema,
      );
      return res.items;
    },
    resolve: (id: string, itemId: string, req: ResolveRequest): Promise<ImportItem> =>
      http.post(
        `/api/v1/imports/${encodeURIComponent(id)}/items/${encodeURIComponent(itemId)}/resolve`,
        'skip' in req ? { skip: true } : { mbid: req.mbid, include_featured: req.includeFeatured },
        itemSchema,
      ),
  };
}

export type ImportApi = ReturnType<typeof createImportApi>;
