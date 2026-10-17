import { z } from 'zod';
import type { HttpClient } from '@/shared/api';
import type { SyncRun } from './model';

const runSchema = z
  .object({
    id: z.string(),
    trigger: z.enum(['schedule', 'manual']),
    started_at: z.string(),
    finished_at: z.string().nullable(),
    artists_total: z.number(),
    artists_done: z.number(),
    new_releases: z.number(),
  })
  .transform((r): SyncRun => ({
    id: r.id,
    trigger: r.trigger,
    startedAt: r.started_at,
    finishedAt: r.finished_at,
    artistsTotal: r.artists_total,
    artistsDone: r.artists_done,
    newReleases: r.new_releases,
  }));

export function createSyncApi(http: HttpClient) {
  return {
    /** null before the very first run. */
    latest: (): Promise<SyncRun | null> =>
      http.get('/api/v1/sync/runs/latest', runSchema.nullable()),
    /** Starts a run over all subscriptions, or returns the one in progress. */
    start: (): Promise<SyncRun> => http.post('/api/v1/sync', undefined, runSchema),
  };
}
