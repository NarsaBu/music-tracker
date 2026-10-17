import { useBackendStatus, type BackendStatus } from '@/entities/system/queries';
import { useTheme } from '@/app/theme/useTheme';

export interface NavItem {
  to: string;
  label: string;
}

export const NAV_ITEMS: readonly NavItem[] = [
  { to: '/', label: 'Feed' },
  { to: '/artists', label: 'Artists' },
  { to: '/imports', label: 'Import' },
  { to: '/settings', label: 'Settings' },
];

const STATUS_TEXT: Record<BackendStatus, string> = {
  checking: 'TUNING…',
  online: 'ONLINE',
  offline: 'OFFLINE',
};

export function useRackLayoutController() {
  const status = useBackendStatus();
  const { theme, setTheme } = useTheme();

  return {
    navItems: NAV_ITEMS,
    statusText: STATUS_TEXT[status],
    statusLed:
      status === 'online'
        ? ('ok' as const)
        : status === 'offline'
          ? ('on' as const)
          : ('warn' as const),
    lampOn: theme === 'night',
    onLampChange: (on: boolean) => setTheme(on ? 'night' : 'day'),
  };
}

export type RackLayoutViewModel = ReturnType<typeof useRackLayoutController>;
