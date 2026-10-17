import { PRIMARY_TYPE_LABELS, PRIMARY_TYPES, SECONDARY_TYPES } from '@/entities/release/model';
import { WEEKDAYS, type DigestFrequency, type Weekday } from '@/entities/settings/model';
import {
  EngravedInput,
  LcdDisplay,
  Led,
  Panel,
  PushButton,
  RotarySelector,
  ToggleSwitch,
} from '@/shared/ui';
import type { SettingsViewModel } from '../controller/useSettingsController';
import styles from './SettingsView.module.css';

const FREQUENCY_OPTIONS = [
  { value: 'daily', label: 'Daily' },
  { value: 'weekly', label: 'Weekly' },
] as const satisfies ReadonlyArray<{ value: DigestFrequency; label: string }>;

const WEEKDAY_OPTIONS = WEEKDAYS.map((d) => ({ value: String(d.value), label: d.short }));

export function SettingsView(vm: SettingsViewModel) {
  const { settings: s, errors } = vm;
  const digest = s.email.digest;

  return (
    <form
      className={styles.layout}
      onSubmit={(e) => {
        e.preventDefault();
        vm.onSave();
      }}
    >
      <Panel title="Типы релизов" className={styles.primary}>
        <div className={styles.switchRow}>
          {PRIMARY_TYPES.map((t) => (
            <ToggleSwitch
              key={t}
              label={PRIMARY_TYPE_LABELS[t]}
              checked={s.filters.primaryTypes.includes(t)}
              onChange={() => vm.onTogglePrimaryType(t)}
            />
          ))}
        </div>
      </Panel>

      <Panel title="Дополнительные типы" className={styles.secondary}>
        <p className={styles.note}>
          Включённые типы попадают в уведомления, выключенные — отсеиваются.
        </p>
        <div className={styles.switchGrid}>
          {SECONDARY_TYPES.map((t) => (
            <ToggleSwitch
              key={t}
              label={t}
              checked={!s.filters.excludedSecondaryTypes.includes(t)}
              onChange={() => vm.onToggleSecondaryType(t)}
            />
          ))}
        </div>
      </Panel>

      <Panel title="Правила" className={styles.rules}>
        <div className={styles.stack}>
          <ToggleSwitch
            label="Featuring"
            description="Релизы, где исполнитель — приглашённый участник. Можно переопределить для каждого исполнителя."
            checked={s.filters.includeFeatured}
            onChange={vm.onIncludeFeaturedChange}
          />
          <ToggleSwitch
            label="Анонсы"
            description="Сообщать о релизах с датой выхода в будущем."
            checked={s.filters.notifyAnnouncements}
            onChange={vm.onNotifyAnnouncementsChange}
          />
        </div>
      </Panel>

      <Panel
        title="Email-дайджест"
        className={styles.email}
        extra={
          <Led
            state={s.email.enabled ? 'ok' : 'off'}
            label={s.email.enabled ? 'Включён' : 'Выключен'}
          />
        }
      >
        <div className={styles.emailGrid}>
          <ToggleSwitch
            label="Отправлять"
            checked={s.email.enabled}
            onChange={vm.onEmailEnabledChange}
          />
          <EngravedInput
            className={styles.emailTo}
            label="Адрес"
            type="email"
            autoComplete="email"
            placeholder="me@example.com"
            value={s.email.to}
            error={errors.emailTo}
            disabled={!s.email.enabled}
            onChange={(e) => vm.onEmailToChange(e.target.value)}
          />
          <RotarySelector
            label="Частота"
            options={FREQUENCY_OPTIONS}
            value={digest.frequency}
            onChange={vm.onFrequencyChange}
            disabled={!s.email.enabled}
          />
          <RotarySelector
            label="День недели"
            options={WEEKDAY_OPTIONS}
            value={String(digest.weekday)}
            onChange={(v) => vm.onWeekdayChange(Number(v) as Weekday)}
            disabled={!s.email.enabled || digest.frequency !== 'weekly'}
          />
          <div className={styles.timeFields}>
            <EngravedInput
              label="Время"
              type="time"
              value={digest.time}
              disabled={!s.email.enabled}
              onChange={(e) => vm.onTimeChange(e.target.value)}
            />
            <EngravedInput
              label="Часовой пояс"
              value={digest.timezone}
              list="timezones"
              disabled={!s.email.enabled}
              onChange={(e) => vm.onTimezoneChange(e.target.value)}
            />
            <datalist id="timezones">
              {Intl.supportedValuesOf('timeZone').map((tz) => (
                <option key={tz} value={tz} />
              ))}
            </datalist>
          </div>
        </div>
      </Panel>

      <Panel title="RSS / Atom" className={styles.rss}>
        <div className={styles.stack}>
          <ToggleSwitch label="Лента" checked={s.rss.enabled} onChange={vm.onRssEnabledChange} />
          <LcdDisplay label="Адрес ленты" size="sm" className={styles.feedUrl}>
            {s.rss.feedUrl || 'NO SIGNAL'}
          </LcdDisplay>
          <PushButton disabled>Перевыпустить ссылку</PushButton>
        </div>
      </Panel>

      <div className={styles.actions}>
        <span className={styles.note}>
          Эталонная страница: изменения пока не сохраняются на сервере.
        </span>
        <PushButton onClick={vm.onReset} disabled={!vm.isDirty}>
          Сбросить
        </PushButton>
        <PushButton type="submit" variant="primary" led={vm.isDirty} disabled={!vm.canSave}>
          Сохранить
        </PushButton>
      </div>
    </form>
  );
}
