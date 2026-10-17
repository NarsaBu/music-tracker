import { useSettingsController } from './controller/useSettingsController';
import { SettingsView } from './view/SettingsView';

export function SettingsPage() {
  const vm = useSettingsController();
  return <SettingsView {...vm} />;
}
