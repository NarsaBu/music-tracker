import { Link } from 'react-router';
import { useI18n } from '@/shared/i18n';
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
import type { ReleasesFeedViewModel } from '../controller/useReleasesFeedController';
import styles from './ReleasesFeedView.module.css';

export function ReleasesFeedView(vm: ReleasesFeedViewModel) {
  const { t: all } = useI18n();
  const t = all.feed;
  return (
    <div className={styles.layout}>
      <Panel
        title={t.title}
        extra={
          <LcdDisplay label={t.count} size="sm">
            {vm.lcdText}
          </LcdDisplay>
        }
      >
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
          <RotarySelector
            label={t.sort}
            options={vm.sortOptions}
            value={vm.sort}
            onChange={vm.onSortChange}
          />
        </div>
      </Panel>

      {vm.isError && (
        <p className={styles.message} role="alert">
          <Led state="warn" />
          {t.loadFailed}
        </p>
      )}

      {vm.isEmpty && (
        <p className={styles.message}>
          {vm.filtered ? (
            t.emptyFiltered
          ) : (
            <>
              {t.emptyBefore}
              <Link to="/artists">{t.emptyLink}</Link>
              {t.emptyAfter}
            </>
          )}
        </p>
      )}

      {vm.records.length > 0 && (
        <RecordShelf label={t.shelf}>
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
                  {r.featured && <TapeLabel>{all.release.feat}</TapeLabel>}
                  {r.announcement && (
                    <TapeLabel tone="accent">{all.release.announcement}</TapeLabel>
                  )}
                </>
              }
            />
          ))}
        </RecordShelf>
      )}

      {vm.hasMore && (
        <div className={styles.more}>
          <PushButton onClick={vm.onLoadMore} disabled={vm.loadingMore} led={vm.loadingMore}>
            {t.loadMore}
          </PushButton>
        </div>
      )}
    </div>
  );
}
