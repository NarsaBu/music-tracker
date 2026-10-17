import { useSyncControlController } from './controller/useSyncControlController';
import { SyncControlView } from './view/SyncControlView';

/** Self-contained widget: pages embed it where a sync control belongs. */
export function SyncControl() {
  const vm = useSyncControlController();
  return <SyncControlView {...vm} />;
}
