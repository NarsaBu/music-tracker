import type { ReactNode } from 'react';
import { Link } from 'react-router';
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
import {
  ROLE_OPTIONS,
  type ArtistCatalogViewModel,
} from '../controller/useArtistCatalogController';
import styles from './ArtistCatalogView.module.css';

export interface ArtistCatalogViewProps extends ArtistCatalogViewModel {
  /** Slot under the header, e.g. the subscription filter editor. */
  filterPanel?: ReactNode;
}

export function ArtistCatalogView({ filterPanel, ...vm }: ArtistCatalogViewProps) {
  if (vm.notSubscribed) {
    return (
      <Panel title="Исполнитель">
        <p>
          На этого исполнителя нет подписки. <Link to="/artists/add">Найти на тюнере</Link>
        </p>
      </Panel>
    );
  }

  return (
    <div className={styles.layout}>
      <Panel
        title={vm.isLoading ? 'Исполнитель' : vm.name}
        extra={
          <LcdDisplay label="Релизов в каталоге" size="sm">
            {vm.recordsLoading ? '---' : `${String(vm.count).padStart(3, '0')} RG`}
          </LcdDisplay>
        }
      >
        <div className={styles.head}>
          <div className={styles.info}>
            {vm.details && <p>{vm.details}</p>}
            <p className={styles.sync}>
              <Led state={vm.checking ? 'warn' : vm.synced ? 'ok' : 'warn'} blink={vm.checking} />
              <span aria-live="polite">{vm.checking ? 'Проверяю MusicBrainz…' : vm.syncText}</span>
            </p>
            {(vm.checkFailed || vm.checkStuck) && (
              <p className={styles.error} role="alert">
                {vm.checkFailed
                  ? 'Не удалось запустить проверку'
                  : 'Проверка затянулась — MusicBrainz может быть недоступен, сервер повторит попытку позже'}
              </p>
            )}
            <PushButton
              variant="primary"
              led={vm.checking}
              disabled={vm.checking}
              onClick={vm.onCheckNow}
            >
              Проверить сейчас
            </PushButton>
          </div>
          <RotarySelector
            label="Роль"
            options={ROLE_OPTIONS}
            value={vm.role}
            onChange={vm.onRoleChange}
          />
        </div>
      </Panel>

      {filterPanel}

      {!vm.recordsLoading && vm.records.length === 0 && (
        <p className={styles.note}>
          {vm.synced
            ? 'Релизов с такой ролью нет.'
            : 'Каталог появится после первой проверки — обычно это занимает несколько секунд.'}
        </p>
      )}

      {vm.records.length > 0 && (
        <RecordShelf label="Каталог релизов">
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
                  {r.featured && <TapeLabel>feat.</TapeLabel>}
                  {r.announcement && <TapeLabel tone="accent">Анонс</TapeLabel>}
                  {r.fresh && <TapeLabel tone="accent">Новое</TapeLabel>}
                </>
              }
            />
          ))}
        </RecordShelf>
      )}

      <p className={styles.back}>
        <Link to="/artists">← К картотеке</Link>
      </p>
    </div>
  );
}
