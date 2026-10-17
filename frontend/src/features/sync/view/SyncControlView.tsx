import { useI18n } from '@/shared/i18n';
import { LcdDisplay, Led, LedBar, Panel, PushButton } from '@/shared/ui';
import type { SyncControlViewModel } from '../controller/useSyncControlController';
import styles from './SyncControlView.module.css';

export function SyncControlView(vm: SyncControlViewModel) {
  const { t: all } = useI18n();
  const t = all.sync;
  return (
    <Panel
      title={t.title}
      extra={
        <LcdDisplay label={t.state} size="sm">
          {vm.lcdText}
        </LcdDisplay>
      }
    >
      <div className={styles.deck}>
        <div className={styles.meter}>
          <LedBar
            className={styles.bar}
            value={vm.progress}
            label={t.progress}
            valueText={vm.progressText}
            active={vm.running}
          />
          <LcdDisplay label={t.artistsChecked} size="sm" className={styles.counter}>
            {vm.counterText}
          </LcdDisplay>
        </div>
        <div className={styles.controls}>
          <p className={styles.status} aria-live="polite">
            {vm.statusText}
          </p>
          {vm.isError && (
            <p className={styles.error} role="alert">
              <Led state="warn" />
              {all.common.serverUnreachable}
            </p>
          )}
          <PushButton
            variant="primary"
            led={vm.running}
            disabled={vm.running || vm.starting}
            onClick={vm.onStart}
          >
            {vm.running ? t.checking : t.checkAll}
          </PushButton>
        </div>
      </div>
    </Panel>
  );
}
