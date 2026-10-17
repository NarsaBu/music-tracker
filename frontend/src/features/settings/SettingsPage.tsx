import { useSettingsQuery } from '@/entities/settings/queries';
import { LcdDisplay, Panel } from '@/shared/ui';
import { useSettingsController } from './controller/useSettingsController';
import type { SettingsState } from '@/entities/settings/model';
import { SettingsView } from './view/SettingsView';

export function SettingsPage() {
  const query = useSettingsQuery();
  if (!query.data) {
    return (
      <Panel title="Настройки">
        <LcdDisplay size="lg" label="Состояние загрузки">
          {query.isError ? 'NO SIGNAL · ERROR' : 'TUNING…'}
        </LcdDisplay>
      </Panel>
    );
  }
  return <SettingsForm state={query.data} />;
}

/** Mounted once the settings are loaded: the controller owns the draft from then on. */
function SettingsForm({ state }: { state: SettingsState }) {
  const vm = useSettingsController(state);
  return <SettingsView {...vm} />;
}
