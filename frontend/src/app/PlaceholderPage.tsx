import { useI18n } from '@/shared/i18n';
import { LcdDisplay, Panel } from '@/shared/ui';

export interface PlaceholderPageProps {
  /** Defaults to "not found". */
  title?: string;
  stage: string;
}

/** Stand-in for screens that do not exist (yet): the 404 page. */
export function PlaceholderPage({ title, stage }: PlaceholderPageProps) {
  const { t } = useI18n();
  return (
    <Panel title={title ?? t.layout.notFound}>
      <LcdDisplay size="lg" label={t.layout.blockStatus}>
        {`${t.common.lcd.noSignal} · ${stage}`}
      </LcdDisplay>
    </Panel>
  );
}
