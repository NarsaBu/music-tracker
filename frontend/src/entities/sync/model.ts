export interface SyncRun {
  id: string;
  trigger: 'schedule' | 'manual';
  startedAt: string;
  finishedAt: string | null;
  artistsTotal: number;
  artistsDone: number;
  /** Release groups found in this run that count as news. */
  newReleases: number;
}

export function isRunning(run: SyncRun | null | undefined): boolean {
  return !!run && run.finishedAt === null;
}

/** 0–100; an empty run is complete. */
export function progressOf(run: SyncRun): number {
  if (run.artistsTotal === 0) return 100;
  return Math.round((run.artistsDone / run.artistsTotal) * 100);
}
