import { z } from 'zod';
import type { HttpClient } from '@/shared/api';

const healthSchema = z.object({ status: z.string() });

export type BackendHealth = z.infer<typeof healthSchema>;

export function createSystemApi(http: HttpClient) {
  return {
    health: () => http.get('/healthz', healthSchema),
  };
}

export type SystemApi = ReturnType<typeof createSystemApi>;
