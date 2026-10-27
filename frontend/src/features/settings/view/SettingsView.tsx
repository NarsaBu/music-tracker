import { PRIMARY_TYPES, SECONDARY_TYPES } from '@/entities/release/model';
import {
  NEW_RELEASE_DAYS_MAX,
  NEW_RELEASE_DAYS_MIN,
  WEEKDAYS,
  type DigestFrequency,
  type Weekday,
} from '@/entities/settings/model';
import { useI18n } from '@/shared/i18n';
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

const FREQUENCIES: readonly DigestFrequency[] = ['daily', 'weekly'];

export function SettingsView(vm: SettingsViewModel) {
  const { t: all } = useI18n();
  const t = all.settings;
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
      <Panel title={t.primaryTypes} className={styles.primary}>
        <div className={styles.switchRow}>
          {PRIMARY_TYPES.map((type) => (
            <ToggleSwitch
              key={type}
              label={all.release.primaryTypes[type]}
              checked={s.filters.primaryTypes.includes(type)}
              onChange={() => vm.onTogglePrimaryType(type)}
            />
          ))}
        </div>
      </Panel>

      <Panel title={t.secondaryTypes} className={styles.secondary}>
        <p className={styles.note}>{t.secondaryNote}</p>
        <div className={styles.switchGrid}>
          {SECONDARY_TYPES.map((type) => (
            <ToggleSwitch
              key={type}
              label={all.release.secondaryTypes[type] ?? type}
              checked={!s.filters.excludedSecondaryTypes.includes(type)}
              onChange={() => vm.onToggleSecondaryType(type)}
            />
          ))}
        </div>
      </Panel>

      <Panel title={t.rules} className={styles.rules}>
        <div className={styles.stack}>
          <ToggleSwitch
            label={all.common.featuring}
            description={t.featuringHint}
            checked={s.filters.includeFeatured}
            onChange={vm.onIncludeFeaturedChange}
          />
          <ToggleSwitch
            label={t.announcements}
            description={t.announcementsHint}
            checked={s.filters.notifyAnnouncements}
            onChange={vm.onNotifyAnnouncementsChange}
          />
          <EngravedInput
            className={styles.days}
            label={t.newReleaseDays}
            hint={t.newReleaseDaysHint}
            error={
              errors.newReleaseDays &&
              t.newReleaseDaysRange(NEW_RELEASE_DAYS_MIN, NEW_RELEASE_DAYS_MAX)
            }
            type="number"
            inputMode="numeric"
            min={NEW_RELEASE_DAYS_MIN}
            max={NEW_RELEASE_DAYS_MAX}
            step={1}
            value={s.filters.newReleaseDays || ''}
            onChange={(e) => vm.onNewReleaseDaysChange(e.target.value)}
          />
        </div>
      </Panel>

      <Panel
        title={t.email}
        className={styles.email}
        extra={
          <Led
            state={s.email.enabled ? 'ok' : 'off'}
            label={s.email.enabled ? t.enabled : t.disabled}
          />
        }
      >
        <div className={styles.emailGrid}>
          <ToggleSwitch
            label={t.send}
            checked={s.email.enabled}
            onChange={vm.onEmailEnabledChange}
          />
          <EngravedInput
            className={styles.emailTo}
            label={t.address}
            type="email"
            autoComplete="email"
            placeholder="me@example.com"
            value={s.email.to}
            error={errors.emailTo && t.invalidEmail}
            disabled={!s.email.enabled}
            onChange={(e) => vm.onEmailToChange(e.target.value)}
          />
          <RotarySelector
            label={t.frequency}
            options={FREQUENCIES.map((f) => ({ value: f, label: t[f] }))}
            value={digest.frequency}
            onChange={vm.onFrequencyChange}
            disabled={!s.email.enabled}
          />
          <RotarySelector
            label={t.weekday}
            options={WEEKDAYS.map((d) => ({ value: String(d), label: t.weekdays[d] ?? '' }))}
            value={String(digest.weekday)}
            onChange={(v) => vm.onWeekdayChange(Number(v) as Weekday)}
            disabled={!s.email.enabled || digest.frequency !== 'weekly'}
          />
          <div className={styles.timeFields}>
            <EngravedInput
              label={t.time}
              type="time"
              value={digest.time}
              disabled={!s.email.enabled}
              onChange={(e) => vm.onTimeChange(e.target.value)}
            />
            <EngravedInput
              label={t.timezone}
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
          <div className={styles.digestStatus}>
            <LcdDisplay label={t.schedule} size="sm">
              {vm.nextDigestText}
            </LcdDisplay>
            {vm.lastDigestText && <span className={styles.note}>{vm.lastDigestText}</span>}
            <PushButton
              onClick={vm.onSendTestEmail}
              disabled={!vm.canTestEmail}
              led={vm.testingEmail}
              title={vm.canTestEmail ? undefined : t.saveAddressFirst}
            >
              {t.testEmail}
            </PushButton>
            {vm.testEmailText && (
              <span
                className={vm.testEmailFailed ? styles.error : styles.note}
                role={vm.testEmailFailed ? 'alert' : 'status'}
              >
                {vm.testEmailText}
              </span>
            )}
          </div>
        </div>
      </Panel>

      <Panel title={t.rss} className={styles.rss}>
        <div className={styles.stack}>
          <ToggleSwitch label={t.feed} checked={s.rss.enabled} onChange={vm.onRssEnabledChange} />
          <p className={styles.note}>{t.rssNote}</p>
          <LcdDisplay label={t.feedUrl} size="sm" className={styles.feedUrl}>
            {s.rss.feedUrl || all.common.lcd.noSignal}
          </LcdDisplay>
          <PushButton onClick={vm.onRotateFeedToken} disabled={vm.rotating} led={vm.rotating}>
            {t.rotate}
          </PushButton>
          {vm.rotateError && (
            <span className={styles.error} role="alert">
              {vm.rotateError}
            </span>
          )}
        </div>
      </Panel>

      <div className={styles.actions}>
        {vm.saveError ? (
          <span className={styles.error} role="alert">
            {vm.saveError}
          </span>
        ) : (
          <span className={styles.note}>{vm.isDirty ? t.dirty : t.saved}</span>
        )}
        <PushButton onClick={vm.onReset} disabled={!vm.isDirty || vm.saving}>
          {t.reset}
        </PushButton>
        <PushButton
          type="submit"
          variant="primary"
          led={vm.isDirty || vm.saving}
          disabled={!vm.canSave}
        >
          {t.save}
        </PushButton>
      </div>
    </form>
  );
}
