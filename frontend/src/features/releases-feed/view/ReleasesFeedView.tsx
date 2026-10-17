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
  TYPE_OPTIONS,
  type ReleasesFeedViewModel,
} from '../controller/useReleasesFeedController';
import styles from './ReleasesFeedView.module.css';

export function ReleasesFeedView(vm: ReleasesFeedViewModel) {
  return (
    <div className={styles.layout}>
      <Panel
        title="Новые релизы"
        extra={
          <LcdDisplay label="Релизов в ленте" size="sm">
            {vm.lcdText}
          </LcdDisplay>
        }
      >
        <div className={styles.filters}>
          <RotarySelector
            label="Тип"
            options={TYPE_OPTIONS}
            value={vm.type}
            onChange={vm.onTypeChange}
          />
          <RotarySelector
            label="Роль"
            options={ROLE_OPTIONS}
            value={vm.role}
            onChange={vm.onRoleChange}
          />
        </div>
      </Panel>

      {vm.isError && (
        <p className={styles.message} role="alert">
          <Led state="warn" />
          Не удалось загрузить ленту
        </p>
      )}

      {vm.isEmpty && (
        <p className={styles.message}>
          {vm.filtered ? (
            'Под этот фильтр ничего не попало.'
          ) : (
            <>
              Новинок пока нет. Они появятся здесь, когда у исполнителей из{' '}
              <Link to="/artists">картотеки</Link> выйдет что-то новое.
            </>
          )}
        </p>
      )}

      {vm.records.length > 0 && (
        <RecordShelf label="Найденные релизы">
          {vm.records.map((r) => (
            <RecordSleeve
              key={r.id}
              title={r.title}
              subtitle={r.credit}
              coverUrl={r.coverUrl}
              href={r.musicbrainzUrl}
              meta={`${r.date} · ${r.foundText}`}
              badges={
                <>
                  {r.typeLabel && <TapeLabel>{r.typeLabel}</TapeLabel>}
                  {r.secondaryTypes.map((t) => (
                    <TapeLabel key={t}>{t}</TapeLabel>
                  ))}
                  {r.featured && <TapeLabel>feat.</TapeLabel>}
                  {r.announcement && <TapeLabel tone="accent">Анонс</TapeLabel>}
                </>
              }
            />
          ))}
        </RecordShelf>
      )}

      {vm.hasMore && (
        <div className={styles.more}>
          <PushButton onClick={vm.onLoadMore} disabled={vm.loadingMore} led={vm.loadingMore}>
            Показать ещё
          </PushButton>
        </div>
      )}
    </div>
  );
}
