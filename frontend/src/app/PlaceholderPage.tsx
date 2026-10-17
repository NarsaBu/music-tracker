import { LcdDisplay, Panel } from '@/shared/ui';

export interface PlaceholderPageProps {
  title: string;
  stage: string;
}

/** Stand-in for screens that arrive in later roadmap stages (docs/09-roadmap.md). */
export function PlaceholderPage({ title, stage }: PlaceholderPageProps) {
  return (
    <Panel title={title}>
      <LcdDisplay size="lg" label="Статус блока">
        {`NO SIGNAL · ${stage}`}
      </LcdDisplay>
    </Panel>
  );
}
