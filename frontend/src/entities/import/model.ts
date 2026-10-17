import type { ArtistCandidate } from '@/entities/artist/model';

export const IMPORT_ITEM_STATUSES = [
  'pending',
  'matched',
  'needs_review',
  'not_found',
  'skipped',
  'failed',
] as const;

export type ImportItemStatus = (typeof IMPORT_ITEM_STATUSES)[number];

/** One uploaded file (US-2). */
export interface ImportJob {
  id: string;
  filename: string;
  total: number;
  counts: Record<ImportItemStatus, number>;
  /** No line is pending any more; some may still wait for review. */
  finished: boolean;
  createdAt: string;
  finishedAt: string | null;
}

/** A search result offered for an ambiguous line; never marked as subscribed. */
export type ImportCandidate = Omit<ArtistCandidate, 'subscribed'>;

export interface ImportItem {
  id: string;
  lineNo: number;
  rawValue: string;
  inputMbid: string | null;
  /** CSV override; null = the global setting. */
  includeFeatured: boolean | null;
  status: ImportItemStatus;
  candidates: ImportCandidate[];
  /** The subscribed artist of a matched line. */
  artist: { mbid: string; name: string } | null;
  error: string | null;
}

export type ResolveRequest = { mbid: string; includeFeatured?: boolean } | { skip: true };

/** Lines the user can still settle by hand (US-3). */
export function isResolvable(status: ImportItemStatus): boolean {
  return status === 'needs_review' || status === 'not_found' || status === 'failed';
}

/** 0–100: share of lines the worker has processed. */
export function progressOf(job: ImportJob): number {
  if (job.total === 0) return 100;
  return Math.round(((job.total - job.counts.pending) / job.total) * 100);
}

/** Lines that wait for the user. */
export function toReviewOf(job: ImportJob): number {
  return job.counts.needs_review + job.counts.not_found + job.counts.failed;
}
