import { useMemo, useState } from 'react';
import type { PrimaryType, SecondaryType } from '@/entities/release/model';
import {
  DEFAULT_SETTINGS,
  toggleItem,
  validateSettings,
  type DigestFrequency,
  type Settings,
  type Weekday,
} from '@/entities/settings/model';

/**
 * Settings screen controller. At stage 0 the state is local (reference page for
 * the design); at stage 3 `initial` and `onSave` come from the settings Model.
 */
export function useSettingsController(initial: Settings = DEFAULT_SETTINGS) {
  const [saved, setSaved] = useState(initial);
  const [draft, setDraft] = useState(initial);

  const errors = useMemo(() => validateSettings(draft), [draft]);
  const isDirty = useMemo(() => JSON.stringify(draft) !== JSON.stringify(saved), [draft, saved]);
  const canSave = isDirty && Object.keys(errors).length === 0;

  const updateFilters = (patch: Partial<Settings['filters']>) =>
    setDraft((s) => ({ ...s, filters: { ...s.filters, ...patch } }));
  const updateEmail = (patch: Partial<Omit<Settings['email'], 'digest'>>) =>
    setDraft((s) => ({ ...s, email: { ...s.email, ...patch } }));
  const updateDigest = (patch: Partial<Settings['email']['digest']>) =>
    setDraft((s) => ({ ...s, email: { ...s.email, digest: { ...s.email.digest, ...patch } } }));

  return {
    settings: draft,
    errors,
    isDirty,
    canSave,

    onTogglePrimaryType: (t: PrimaryType) =>
      updateFilters({ primaryTypes: toggleItem(draft.filters.primaryTypes, t) }),
    // The UI shows "include" switches; the model stores exclusions.
    onToggleSecondaryType: (t: SecondaryType) =>
      updateFilters({
        excludedSecondaryTypes: toggleItem(draft.filters.excludedSecondaryTypes, t),
      }),
    onIncludeFeaturedChange: (v: boolean) => updateFilters({ includeFeatured: v }),
    onNotifyAnnouncementsChange: (v: boolean) => updateFilters({ notifyAnnouncements: v }),

    onEmailEnabledChange: (v: boolean) => updateEmail({ enabled: v }),
    onEmailToChange: (v: string) => updateEmail({ to: v }),
    onFrequencyChange: (v: DigestFrequency) => updateDigest({ frequency: v }),
    onWeekdayChange: (v: Weekday) => updateDigest({ weekday: v }),
    onTimeChange: (v: string) => updateDigest({ time: v }),
    onTimezoneChange: (v: string) => updateDigest({ timezone: v }),

    onRssEnabledChange: (v: boolean) => setDraft((s) => ({ ...s, rss: { ...s.rss, enabled: v } })),

    onSave: () => {
      if (canSave) setSaved(draft);
    },
    onReset: () => setDraft(saved),
  };
}

export type SettingsViewModel = ReturnType<typeof useSettingsController>;
