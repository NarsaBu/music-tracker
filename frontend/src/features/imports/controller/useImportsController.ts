import { useState } from 'react';
import { progressOf, toReviewOf, type ImportJob } from '@/entities/import/model';
import { useImportsQuery, useUploadImportMutation } from '@/entities/import/queries';
import type { LedState } from '@/shared/ui';
import { ApiError } from '@/shared/api';
import { useI18n, type Messages } from '@/shared/i18n';
import { formatDateTime, lcdPair } from '@/shared/lib/format';

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

function toRow(job: ImportJob, all: Messages): ImportRow {
  const t = all.imports;
  const toReview = toReviewOf(job);
  const done = job.total - job.counts.pending;
  const parts = [t.added(job.counts.matched)];
  if (toReview) parts.push(t.toReview(toReview));
  if (job.counts.skipped) parts.push(t.skipped(job.counts.skipped));
  return {
    id: job.id,
    filename: job.filename,
    createdText: formatDateTime(job.createdAt, all.locale),
    progress: progressOf(job),
    counterText: lcdPair(done, job.total),
    led: !job.finished ? 'on' : toReview ? 'warn' : 'ok',
    processing: !job.finished,
    statusText: (job.finished ? '' : t.matching) + parts.join(' · '),
    toReview,
  };
}

function uploadErrorText(error: unknown, t: Messages['imports']): string {
  if (error instanceof ApiError && error.status === 422) {
    return t.rejected(error.fields?.[0]?.message ?? error.detail ?? '');
  }
  return t.uploadFailed;
}

/** Import screen ("cassette deck", US-2): upload a file, watch the imports progress. */
export function useImportsController() {
  const { t: all } = useI18n();
  const t = all.imports;
  const [tapeLabel, setTapeLabel] = useState<string>();
  const [localError, setLocalError] = useState<string>();
  const imports = useImportsQuery();
  const upload = useUploadImportMutation();

  const rows = (imports.data ?? []).map((job) => toRow(job, all));
  const playing = rows.find((r) => r.processing);

  const lcdText = upload.isPending
    ? t.lcd.loading
    : playing
      ? t.lcd.play(playing.counterText)
      : rows.length
        ? t.lcd.stop
        : t.lcd.insert;

  return {
    lcdText,
    tapeLabel,
    uploading: upload.isPending,
    /** The reels spin while a file is uploaded or matched. */
    spinning: upload.isPending || !!playing,
    uploadError: localError ?? (upload.isError ? uploadErrorText(upload.error, t) : undefined),
    rows,
    isLoading: imports.isPending,
    isError: imports.isError,
    onFile: (file: File) => {
      setTapeLabel(file.name);
      upload.reset();
      // Checked here too, so a big file is not sent only to be rejected.
      if (file.size > MAX_FILE_BYTES) {
        setLocalError(t.tooBig);
        return;
      }
      setLocalError(undefined);
      upload.mutate(file);
    },
  };
}

export type ImportsViewModel = ReturnType<typeof useImportsController>;
