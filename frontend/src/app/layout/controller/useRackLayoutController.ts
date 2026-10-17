import { useBackendStatus } from '@/entities/system/queries';
import { useTheme } from '@/app/theme/useTheme';
import { LANGS, useI18n, type Lang, type Messages } from '@/shared/i18n';

export interface NavItem {
  to: string;
  label: string;
}

const NAV: ReadonlyArray<{ to: string; key: keyof Messages['layout']['nav'] }> = [
  { to: '/', key: 'feed' },
  { to: '/artists', key: 'artists' },
  { to: '/imports', key: 'imports' },
  { to: '/settings', key: 'settings' },
];

export function useRackLayoutController() {
  const status = useBackendStatus();
  const { theme, setTheme } = useTheme();
  const { lang, setLang, t } = useI18n();

  return {
    t: t.layout,
    navItems: NAV.map((n): NavItem => ({ to: n.to, label: t.layout.nav[n.key] })),
    statusText: t.layout.status[status],
    statusLed:
      status === 'online'
        ? ('ok' as const)
        : status === 'offline'
          ? ('on' as const)
          : ('warn' as const),
    // The lamp lights the room: on is the day theme.
    lampOn: theme === 'day',
    onLampChange: (on: boolean) => setTheme(on ? 'day' : 'night'),
    langs: LANGS,
    lang,
    onLangChange: (next: Lang) => setLang(next),
  };
}

export type RackLayoutViewModel = ReturnType<typeof useRackLayoutController>;
