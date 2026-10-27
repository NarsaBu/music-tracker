import { useMemo, useState } from 'react';
import type { PrimaryType, SecondaryType } from '@/entities/release/model';
import {
  toggleItem,
  validateSettings,
  type DigestFrequency,
  type Settings,
  type SettingsState,
  type Weekday,
} from '@/entities/settings/model';
import {
  useRotateFeedTokenMutation,
  useSaveSettingsMutation,
  useSendTestEmailMutation,
} from '@/entities/settings/queries';
import { ApiError } from '@/shared/api';
import { useI18n, type Messages } from '@/shared/i18n';
import { formatDateTime } from '@/shared/lib/format';

function errorText(error: unknown, fallback: string, t: Messages['settings']): string {
  if (error instanceof ApiError) {
    if (error.status === 503) return t.smtpDown;
    if (error.fields.length > 0)
      return error.fields.map((f) => `${f.field}: ${f.message}`).join('; ');
  }
  return fallback;
}

/** Settings screen ("amplifier panel", US-5, US-7, US-8): edit, save, test email, rotate feed. */
export function useSettingsController(initial: SettingsState) {
  const { t: all } = useI18n();
  const t = all.settings;
  const at = (iso: string) => formatDateTime(iso, all.locale);
  const [saved, setSaved] = useState(initial.settings);
  const [draft, setDraft] = useState(initial.settings);
  const [schedule, setSchedule] = useState({
    last: initial.lastDigestAt,
    next: initial.nextDigestAt,
  });

  const save = useSaveSettingsMutation();
  const testEmail = useSendTestEmailMutation();
  const rotate = useRotateFeedTokenMutation();

  const errors = useMemo(() => validateSettings(draft), [draft]);
  const isDirty = useMemo(() => JSON.stringify(draft) !== JSON.stringify(saved), [draft, saved]);
  const canSave = isDirty && Object.keys(errors).length === 0 && !save.isPending;
  // The test goes to the saved address, so unsaved email edits block it.
  const emailDirty = JSON.stringify(draft.email) !== JSON.stringify(saved.email);

  const updateFilters = (patch: Partial<Settings['filters']>) =>
    setDraft((s) => ({ ...s, filters: { ...s.filters, ...patch } }));
  const updateEmail = (patch: Partial<Omit<Settings['email'], 'digest'>>) =>
    setDraft((s) => ({ ...s, email: { ...s.email, ...patch } }));
  const updateDigest = (patch: Partial<Settings['email']['digest']>) =>
    setDraft((s) => ({ ...s, email: { ...s.email, digest: { ...s.email.digest, ...patch } } }));

  let testEmailText: string | undefined;
  if (testEmail.isSuccess) testEmailText = t.testSent(saved.email.to);
  if (testEmail.isError) testEmailText = errorText(testEmail.error, t.testFailed, t);

  return {
    settings: draft,
    errors,
    isDirty,
    canSave,
    saving: save.isPending,
    saveError: save.isError ? errorText(save.error, t.saveFailed, t) : undefined,
    nextDigestText: schedule.next ? t.nextDigest(at(schedule.next)) : t.digestOff,
    lastDigestText: schedule.last ? t.lastDigest(at(schedule.last)) : undefined,
    canTestEmail: saved.email.to !== '' && !emailDirty && !testEmail.isPending,
    testingEmail: testEmail.isPending,
    testEmailText,
    testEmailFailed: testEmail.isError,
    rotating: rotate.isPending,
    rotateError: rotate.isError ? t.rotateFailed : undefined,

    onTogglePrimaryType: (t: PrimaryType) =>
      updateFilters({ primaryTypes: toggleItem(draft.filters.primaryTypes, t) }),
    // The UI shows "include" switches; the model stores exclusions.
    onToggleSecondaryType: (t: SecondaryType) =>
      updateFilters({
        excludedSecondaryTypes: toggleItem(draft.filters.excludedSecondaryTypes, t),
      }),
    onIncludeFeaturedChange: (v: boolean) => updateFilters({ includeFeatured: v }),
    onNotifyAnnouncementsChange: (v: boolean) => updateFilters({ notifyAnnouncements: v }),
    // An empty field is 0 and fails validation until a number is typed.
    onNewReleaseDaysChange: (v: string) => updateFilters({ newReleaseDays: Number(v) }),

    onEmailEnabledChange: (v: boolean) => updateEmail({ enabled: v }),
    onEmailToChange: (v: string) => updateEmail({ to: v }),
    onFrequencyChange: (v: DigestFrequency) => updateDigest({ frequency: v }),
    onWeekdayChange: (v: Weekday) => updateDigest({ weekday: v }),
    onTimeChange: (v: string) => updateDigest({ time: v }),
    onTimezoneChange: (v: string) => updateDigest({ timezone: v }),

    onRssEnabledChange: (v: boolean) => setDraft((s) => ({ ...s, rss: { ...s.rss, enabled: v } })),

    onSave: () => {
      if (!canSave) return;
      testEmail.reset();
      save.mutate(draft, {
        onSuccess: (state) => {
          setSaved(state.settings);
          setDraft(state.settings);
          setSchedule({ last: state.lastDigestAt, next: state.nextDigestAt });
        },
      });
    },
    onReset: () => setDraft(saved),
    onSendTestEmail: () => testEmail.mutate(),
    onRotateFeedToken: () =>
      rotate.mutate(undefined, {
        // The URL is not a user edit: update both copies so the form stays clean.
        onSuccess: (feedUrl) => {
          setSaved((s) => ({ ...s, rss: { ...s.rss, feedUrl } }));
          setDraft((s) => ({ ...s, rss: { ...s.rss, feedUrl } }));
        },
      }),
  };
}

export type SettingsViewModel = ReturnType<typeof useSettingsController>;
