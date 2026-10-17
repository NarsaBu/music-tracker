import { useSettingsQuery } from '@/entities/settings/queries';
import { useI18n } from '@/shared/i18n';
import { LcdDisplay, Panel } from '@/shared/ui';
import { useSettingsController } from './controller/useSettingsController';
import type { SettingsState } from '@/entities/settings/model';
import { SettingsView } from './view/SettingsView';

export function SettingsPage() {
  const query = useSettingsQuery();
  const { t } = useI18n();
  if (!query.data) {
    return (
      <Panel title={t.settings.title}>
        <LcdDisplay size="lg" label={t.settings.loading}>
          {query.isError ? t.common.lcd.loadError : t.common.lcd.tuning}
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
