import type { ReactNode } from 'react';
import { Link } from 'react-router';
import { useI18n } from '@/shared/i18n';
import { LCD_BLANK, lcdCount } from '@/shared/lib/format';
import {
  LcdDisplay,
  Led,
  Panel,
  PushButton,
  RecordShelf,
  RecordSleeve,
  RotarySelector,
  TapeLabel,
} from '@/shared/ui';
import type { ArtistCatalogViewModel } from '../controller/useArtistCatalogController';
import styles from './ArtistCatalogView.module.css';

export interface ArtistCatalogViewProps extends ArtistCatalogViewModel {
  /** Slot under the header, e.g. the subscription filter editor. */
  filterPanel?: ReactNode;
}

export function ArtistCatalogView({ filterPanel, ...vm }: ArtistCatalogViewProps) {
  const { t: all } = useI18n();
  const t = all.catalog;
  if (vm.notSubscribed) {
    return (
      <Panel title={t.artist}>
        <p>
          {t.notSubscribed} <Link to="/artists/add">{t.findOnTuner}</Link>
        </p>
      </Panel>
    );
  }

  return (
    <div className={styles.layout}>
      <Panel
        title={vm.isLoading ? t.artist : vm.name}
        extra={
          <LcdDisplay label={t.count} size="sm">
            {vm.recordsLoading ? LCD_BLANK : lcdCount(vm.count)}
          </LcdDisplay>
        }
      >
        <div className={styles.head}>
          <div className={styles.info}>
            {vm.details && <p>{vm.details}</p>}
            <p className={styles.sync}>
              <Led state={vm.checking ? 'warn' : vm.synced ? 'ok' : 'warn'} blink={vm.checking} />
              <span aria-live="polite">{vm.checking ? t.checking : vm.syncText}</span>
            </p>
            {(vm.checkFailed || vm.checkStuck) && (
              <p className={styles.error} role="alert">
                {vm.checkFailed ? t.checkFailed : t.checkStuck}
              </p>
            )}
            <PushButton
              variant="primary"
              led={vm.checking}
              disabled={vm.checking}
              onClick={vm.onCheckNow}
            >
              {t.checkNow}
            </PushButton>
          </div>
          <div className={styles.filters}>
            <RotarySelector
              label={t.type}
              options={vm.typeOptions}
              value={vm.type}
              onChange={vm.onTypeChange}
            />
            <RotarySelector
              label={t.role}
              options={vm.roleOptions}
              value={vm.role}
              onChange={vm.onRoleChange}
            />
          </div>
        </div>
      </Panel>

      {filterPanel}

      {!vm.recordsLoading && vm.records.length === 0 && (
        <p className={styles.note}>
          {vm.synced ? (vm.filtered ? t.noMatch : t.empty) : t.notSynced}
        </p>
      )}

      {vm.records.length > 0 && (
        <RecordShelf label={t.shelf}>
          {vm.records.map((r) => (
            <RecordSleeve
              key={r.mbid}
              title={r.title}
              subtitle={r.credit}
              coverUrl={r.coverUrl}
              href={r.musicbrainzUrl}
              meta={r.date}
              badges={
                <>
                  {r.typeLabel && <TapeLabel>{r.typeLabel}</TapeLabel>}
                  {r.secondaryTypes.map((t) => (
                    <TapeLabel key={t}>{t}</TapeLabel>
                  ))}
                  {r.featured && <TapeLabel>{all.release.feat}</TapeLabel>}
                  {r.announcement && (
                    <TapeLabel tone="accent">{all.release.announcement}</TapeLabel>
                  )}
                  {r.fresh && <TapeLabel tone="accent">{all.release.fresh}</TapeLabel>}
                </>
              }
            />
          ))}
        </RecordShelf>
      )}

      <p className={styles.back}>
        <Link to="/artists">{t.back}</Link>
      </p>
    </div>
  );
}
