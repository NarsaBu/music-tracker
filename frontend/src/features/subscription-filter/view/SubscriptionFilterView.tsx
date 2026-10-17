import { PRIMARY_TYPE_LABELS, PRIMARY_TYPES, SECONDARY_TYPES } from '@/entities/release/model';
import { Led, Panel, ToggleSwitch } from '@/shared/ui';
import type { SubscriptionFilterViewModel } from '../controller/useSubscriptionFilterController';
import styles from './SubscriptionFilterView.module.css';

export function SubscriptionFilterView(vm: SubscriptionFilterViewModel) {
  if (!vm.ready) return null;
  const busy = vm.saving;

  return (
    <Panel
      title="Фильтр уведомлений"
      extra={
        <Led state={busy ? 'warn' : 'ok'} blink={busy} label={busy ? 'Сохраняю' : 'Сохранено'} />
      }
    >
      <p className={styles.note}>
        По умолчанию действуют общие настройки. Включите «Своё», чтобы задать значение только для
        этого исполнителя.
      </p>
      {vm.error && (
        <p className={styles.error} role="alert">
          {vm.error}
        </p>
      )}

      <fieldset className={styles.row} disabled={busy}>
        <legend className={styles.legend}>Типы релизов</legend>
        <ToggleSwitch
          label="Своё"
          checked={vm.ownTypes}
          onChange={vm.onOwnTypesChange}
          withLed={false}
        />
        <div className={styles.switches}>
          {PRIMARY_TYPES.map((t) => (
            <ToggleSwitch
              key={t}
              label={PRIMARY_TYPE_LABELS[t]}
              checked={vm.primaryTypes.includes(t)}
              disabled={!vm.ownTypes}
              onChange={() => vm.onTogglePrimaryType(t)}
            />
          ))}
        </div>
      </fieldset>

      <fieldset className={styles.row} disabled={busy}>
        <legend className={styles.legend}>Дополнительные типы</legend>
        <ToggleSwitch
          label="Своё"
          checked={vm.ownSecondary}
          onChange={vm.onOwnSecondaryChange}
          withLed={false}
        />
        <div className={styles.switches}>
          {SECONDARY_TYPES.map((t) => (
            <ToggleSwitch
              key={t}
              label={t}
              checked={!vm.excludedSecondaryTypes.includes(t)}
              disabled={!vm.ownSecondary}
              onChange={() => vm.onToggleSecondaryType(t)}
            />
          ))}
        </div>
      </fieldset>

      <fieldset className={styles.row} disabled={busy}>
        <legend className={styles.legend}>Featuring</legend>
        <ToggleSwitch
          label="Своё"
          checked={vm.ownFeatured}
          onChange={vm.onOwnFeaturedChange}
          withLed={false}
        />
        <ToggleSwitch
          label="Гостевые релизы"
          description="Сообщать о релизах, где исполнитель — приглашённый участник."
          checked={vm.includeFeatured}
          disabled={!vm.ownFeatured}
          onChange={vm.onIncludeFeaturedChange}
        />
      </fieldset>
    </Panel>
  );
}
