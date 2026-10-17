import type { ReactNode } from 'react';
import { PRIMARY_TYPES, SECONDARY_TYPES } from '@/entities/release/model';
import { useI18n } from '@/shared/i18n';
import { Led, Panel, PushButton } from '@/shared/ui';
import type { SubscriptionFilterViewModel } from '../controller/useSubscriptionFilterController';
import styles from './SubscriptionFilterView.module.css';

/**
 * Compact per-artist filter: one row per field — an "own" key that takes the
 * field over from the global settings, then latching keys for the values.
 */
export function SubscriptionFilterView(vm: SubscriptionFilterViewModel) {
  const { t: all } = useI18n();
  const t = all.filter;
  if (!vm.ready) return null;
  const busy = vm.saving;

  return (
    <Panel
      title={t.title}
      extra={<Led state={busy ? 'warn' : 'ok'} blink={busy} label={busy ? t.saving : t.saved} />}
    >
      <p className={styles.note}>{t.note}</p>
      {vm.error && (
        <p className={styles.error} role="alert">
          {vm.error}
        </p>
      )}

      <Row legend={t.types} own={vm.ownTypes} onOwnChange={vm.onOwnTypesChange} busy={busy}>
        {PRIMARY_TYPES.map((type) => (
          <Key
            key={type}
            label={all.release.primaryTypes[type]}
            on={vm.primaryTypes.includes(type)}
            disabled={!vm.ownTypes}
            onClick={() => vm.onTogglePrimaryType(type)}
          />
        ))}
      </Row>

      <Row
        legend={t.secondary}
        own={vm.ownSecondary}
        onOwnChange={vm.onOwnSecondaryChange}
        busy={busy}
      >
        {SECONDARY_TYPES.map((type) => (
          <Key
            key={type}
            label={all.release.secondaryTypes[type] ?? type}
            on={!vm.excludedSecondaryTypes.includes(type)}
            disabled={!vm.ownSecondary}
            onClick={() => vm.onToggleSecondaryType(type)}
          />
        ))}
      </Row>

      <Row
        legend={all.common.featuring}
        own={vm.ownFeatured}
        onOwnChange={vm.onOwnFeaturedChange}
        busy={busy}
      >
        <Key
          label={t.guestReleases}
          title={all.common.featuringHint}
          on={vm.includeFeatured}
          disabled={!vm.ownFeatured}
          onClick={() => vm.onIncludeFeaturedChange(!vm.includeFeatured)}
        />
      </Row>
    </Panel>
  );
}

function Row({
  legend,
  own,
  onOwnChange,
  busy,
  children,
}: {
  legend: string;
  own: boolean;
  onOwnChange: (own: boolean) => void;
  busy: boolean;
  children: ReactNode;
}) {
  const { t } = useI18n();
  return (
    <fieldset className={styles.row} disabled={busy}>
      <legend className={styles.legend}>{legend}</legend>
      <PushButton
        size="sm"
        variant="primary"
        className={styles.own}
        led={own}
        aria-pressed={own}
        onClick={() => onOwnChange(!own)}
      >
        {t.filter.own}
      </PushButton>
      <div className={styles.keys}>{children}</div>
    </fieldset>
  );
}

function Key({
  label,
  on,
  disabled,
  title,
  onClick,
}: {
  label: string;
  on: boolean;
  disabled: boolean;
  title?: string;
  onClick: () => void;
}) {
  return (
    <PushButton
      size="sm"
      led={on}
      aria-pressed={on}
      disabled={disabled}
      title={title}
      onClick={onClick}
    >
      {label}
    </PushButton>
  );
}
