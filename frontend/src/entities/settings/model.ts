import type { PrimaryType, SecondaryType } from '@/entities/release/model';

export type DigestFrequency = 'daily' | 'weekly';
/** 0 = Sunday … 6 = Saturday, same as the backend. */
export type Weekday = 0 | 1 | 2 | 3 | 4 | 5 | 6;

export interface ReleaseFilterSettings {
  primaryTypes: PrimaryType[];
  excludedSecondaryTypes: SecondaryType[];
  includeFeatured: boolean;
  notifyAnnouncements: boolean;
  /** A release older than this many days, found only now, is not news. */
  newReleaseDays: number;
}

export interface DigestSettings {
  frequency: DigestFrequency;
  weekday: Weekday;
  /** HH:MM in `timezone`. */
  time: string;
  timezone: string;
}

export interface Settings {
  filters: ReleaseFilterSettings;
  email: { enabled: boolean; to: string; digest: DigestSettings };
  rss: { enabled: boolean; feedUrl: string };
}

/** Settings as loaded from the server, with the read-only digest schedule. */
export interface SettingsState {
  settings: Settings;
  lastDigestAt: string | null;
  /** null while email is off. */
  nextDigestAt: string | null;
}

/** Mirrors the database defaults (backend/migrations/000001_init.up.sql). */
export const DEFAULT_SETTINGS: Settings = {
  filters: {
    primaryTypes: ['album', 'ep', 'single'],
    excludedSecondaryTypes: [
      'Compilation',
      'DJ-mix',
      'Interview',
      'Spokenword',
      'Audiobook',
      'Audio drama',
      'Demo',
      'Field recording',
    ],
    includeFeatured: false,
    notifyAnnouncements: true,
    newReleaseDays: 90,
  },
  email: {
    enabled: false,
    to: '',
    digest: { frequency: 'daily', weekday: 5, time: '09:00', timezone: 'UTC' },
  },
  rss: { enabled: true, feedUrl: '' },
};

/** Weekdays in the order of the selector: Monday first. */
export const WEEKDAYS: readonly Weekday[] = [1, 2, 3, 4, 5, 6, 0];

/** Same bounds as the backend (domain.MinNewReleaseDays…MaxNewReleaseDays). */
export const NEW_RELEASE_DAYS_MIN = 1;
export const NEW_RELEASE_DAYS_MAX = 3650;

const EMAIL_RE = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

/** Error codes; the view turns them into text. */
export interface SettingsErrors {
  emailTo?: 'invalid';
  newReleaseDays?: 'range';
}

/** Business validation shared by every form that edits Settings. */
export function validateSettings(s: Settings): SettingsErrors {
  const errors: SettingsErrors = {};
  const days = s.filters.newReleaseDays;
  if (!Number.isInteger(days) || days < NEW_RELEASE_DAYS_MIN || days > NEW_RELEASE_DAYS_MAX) {
    errors.newReleaseDays = 'range';
  }
  if (s.email.enabled && !EMAIL_RE.test(s.email.to.trim())) {
    errors.emailTo = 'invalid';
  }
  return errors;
}

/** Adds the item when absent, removes it when present. */
export function toggleItem<T>(list: readonly T[], item: T): T[] {
  return list.includes(item) ? list.filter((x) => x !== item) : [...list, item];
}
