import { useQueryClient } from '@tanstack/react-query';
import { useEffect, useRef } from 'react';
import { releaseKeys } from '@/entities/release/queries';
import { subscriptionKeys } from '@/entities/subscription/queries';
import { isRunning, progressOf } from '@/entities/sync/model';
import { useLatestSyncRunQuery, useStartSyncMutation } from '@/entities/sync/queries';
import { formatDateTime } from '@/shared/lib/format';

/** Whole-library sync ("Play" button + VU meter, US-10). */
export function useSyncControlController() {
  const queryClient = useQueryClient();
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

  let lcdText = 'NO RUNS';
  let statusText = 'Проверок ещё не было';
  if (run && running) {
    lcdText = `${run.artistsDone}/${run.artistsTotal}`;
    statusText = `Идёт проверка: ${run.artistsDone} из ${run.artistsTotal}`;
  } else if (run?.finishedAt) {
    lcdText = `NEW ${String(run.newReleases).padStart(3, '0')}`;
    statusText = `Последняя проверка: ${formatDateTime(run.finishedAt)} · новинок: ${run.newReleases}`;
  }

  return {
    running,
    progress: run && running ? progressOf(run) : 0,
    lcdText,
    statusText,
    starting: start.isPending,
    isError: latest.isError || start.isError,
    onStart: () => start.mutate(),
  };
}

export type SyncControlViewModel = ReturnType<typeof useSyncControlController>;
