import { LcdDisplay, Led, Panel, PushButton, VuMeter } from '@/shared/ui';
import type { SyncControlViewModel } from '../controller/useSyncControlController';
import styles from './SyncControlView.module.css';

export function SyncControlView(vm: SyncControlViewModel) {
  return (
    <Panel
      title="Синхронизация"
      extra={
        <LcdDisplay label="Состояние синхронизации" size="sm">
          {vm.lcdText}
        </LcdDisplay>
      }
    >
      <div className={styles.deck}>
        <VuMeter value={vm.progress} label="Прогресс синхронизации" />
        <div className={styles.controls}>
          <p className={styles.status} aria-live="polite">
            {vm.statusText}
          </p>
          {vm.isError && (
            <p className={styles.error} role="alert">
              <Led state="warn" />
              Не удалось связаться с сервером
            </p>
          )}
          <PushButton
            variant="primary"
            led={vm.running}
            disabled={vm.running || vm.starting}
            onClick={vm.onStart}
          >
            {vm.running ? 'Проверяю…' : 'Проверить все'}
          </PushButton>
        </div>
      </div>
    </Panel>
  );
}
