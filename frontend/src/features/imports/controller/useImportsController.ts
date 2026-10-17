import { useState } from 'react';
import { progressOf, toReviewOf, type ImportJob } from '@/entities/import/model';
import { useImportsQuery, useUploadImportMutation } from '@/entities/import/queries';
import type { LedState } from '@/shared/ui';
import { ApiError } from '@/shared/api';
import { formatDateTime } from '@/shared/lib/format';

export const ACCEPTED_FILES = '.txt,.csv,text/plain,text/csv';
export const MAX_FILE_BYTES = 1 << 20;

export interface ImportRow {
  id: string;
  filename: string;
  createdText: string;
  progress: number;
  /** Tape counter: processed / total lines. */
  counterText: string;
  led: LedState;
  processing: boolean;
  statusText: string;
  /** Lines that wait for the user. */
  toReview: number;
}

const pad = (n: number) => String(n).padStart(3, '0');

function toRow(job: ImportJob): ImportRow {
  const toReview = toReviewOf(job);
  const done = job.total - job.counts.pending;
  const parts = [`добавлено ${job.counts.matched}`];
  if (toReview) parts.push(`ждут разбора ${toReview}`);
  if (job.counts.skipped) parts.push(`пропущено ${job.counts.skipped}`);
  return {
    id: job.id,
    filename: job.filename,
    createdText: formatDateTime(job.createdAt),
    progress: progressOf(job),
    counterText: `${pad(done)}/${pad(job.total)}`,
    led: !job.finished ? 'on' : toReview ? 'warn' : 'ok',
    processing: !job.finished,
    statusText: (job.finished ? '' : 'Идёт сопоставление · ') + parts.join(' · '),
    toReview,
  };
}

function uploadErrorText(error: unknown): string {
  if (error instanceof ApiError && error.status === 422) {
    const detail = error.fields?.[0]?.message ?? error.detail;
    return `Файл не принят${detail ? `: ${detail}` : ''}`;
  }
  return 'Не удалось загрузить файл';
}

/** Import screen ("cassette deck", US-2): upload a file, watch the imports progress. */
export function useImportsController() {
  const [tapeLabel, setTapeLabel] = useState<string>();
  const [localError, setLocalError] = useState<string>();
  const imports = useImportsQuery();
  const upload = useUploadImportMutation();

  const rows = (imports.data ?? []).map(toRow);
  const playing = rows.find((r) => r.processing);

  const lcdText = upload.isPending
    ? 'LOADING…'
    : playing
      ? `PLAY ${playing.counterText}`
      : rows.length
        ? 'STOP'
        : 'INSERT TAPE';

  return {
    lcdText,
    tapeLabel,
    uploading: upload.isPending,
    /** The reels spin while a file is uploaded or matched. */
    spinning: upload.isPending || !!playing,
    uploadError: localError ?? (upload.isError ? uploadErrorText(upload.error) : undefined),
    rows,
    isLoading: imports.isPending,
    isError: imports.isError,
    onFile: (file: File) => {
      setTapeLabel(file.name);
      upload.reset();
      // Checked here too, so a big file is not sent only to be rejected.
      if (file.size > MAX_FILE_BYTES) {
        setLocalError('Файл больше 1 МБ — разбейте список на части');
        return;
      }
      setLocalError(undefined);
      upload.mutate(file);
    },
  };
}

export type ImportsViewModel = ReturnType<typeof useImportsController>;
