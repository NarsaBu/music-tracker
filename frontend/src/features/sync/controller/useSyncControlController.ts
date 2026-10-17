import { useQueryClient } from '@tanstack/react-query';
import { useEffect, useRef } from 'react';
import { releaseKeys } from '@/entities/release/queries';
import { subscriptionKeys } from '@/entities/subscription/queries';
import { isRunning, progressOf } from '@/entities/sync/model';
import { useLatestSyncRunQuery, useStartSyncMutation } from '@/entities/sync/queries';
import { useI18n } from '@/shared/i18n';
import { formatDateTime, LCD_BLANK, lcdCount, lcdPair } from '@/shared/lib/format';

/** Whole-library sync ("Play" button + LED progress bar, US-10). */
export function useSyncControlController() {
  const queryClient = useQueryClient();
  const { t: all } = useI18n();
  const t = all.sync;
  const latest = useLatestSyncRunQuery();
  const start = useStartSyncMutation();
  const run = latest.data ?? null;
  const running = isRunning(run);

  // When a run finishes, cards and catalogs show stale sync times: refresh them.
  const wasRunning = useRef(running);
  useEffect(() => {
    if (wasRunning.current && !running) {
      void queryClient.invalidateQueries({ queryKey: subscriptionKeys.all });
      void queryClient.invalidateQueries({ queryKey: releaseKeys.all });
    }
    wasRunning.current = running;
  }, [running, queryClient]);

  // A finished run stays on the bar as a full one, so the last result is visible.
  const progress = run ? (running ? progressOf(run) : 100) : 0;
  let lcdText = t.noRunsLcd;
  let statusText = t.noRuns;
  if (run && running) {
    lcdText = `${String(progress).padStart(3, '0')}%`;
    statusText = t.running(run.artistsDone, run.artistsTotal);
  } else if (run?.finishedAt) {
    lcdText = t.newLcd(lcdCount(run.newReleases));
    statusText = t.finished(formatDateTime(run.finishedAt, all.locale), run.newReleases);
  }

  return {
    running,
    progress,
    /** Tape-counter style "120/712"; dashes before the first run. */
    counterText: run ? lcdPair(run.artistsDone, run.artistsTotal) : `${LCD_BLANK}/${LCD_BLANK}`,
    progressText: run ? t.counter(run.artistsDone, run.artistsTotal) : t.noRuns,
    lcdText,
    statusText,
    starting: start.isPending,
    isError: latest.isError || start.isError,
    onStart: () => start.mutate(),
  };
}

export type SyncControlViewModel = ReturnType<typeof useSyncControlController>;
