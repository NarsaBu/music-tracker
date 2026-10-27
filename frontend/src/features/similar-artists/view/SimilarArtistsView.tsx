import { Link } from 'react-router';
import { useI18n } from '@/shared/i18n';
import { Led, Panel, PushButton, SignalMeter } from '@/shared/ui';
import type { SimilarArtistsViewModel } from '../controller/useSimilarArtistsController';
import styles from './SimilarArtistsView.module.css';

export function SimilarArtistsView(vm: SimilarArtistsViewModel) {
  const { t: all } = useI18n();
  const t = all.similar;
  return (
    <Panel title={t.title} extra={<span className={styles.source}>{t.source}</span>}>
      {vm.isLoading && <p className={styles.note}>{t.loading}</p>}

      {vm.errorText && (
        <p className={styles.error} role="alert">
          <Led state="warn" />
          {vm.errorText}
          {vm.loadFailed && <PushButton onClick={vm.onRetry}>{t.retry}</PushButton>}
        </p>
      )}

      {vm.isEmpty && <p className={styles.note}>{t.empty}</p>}

      {vm.rows.length > 0 && (
        <ul className={styles.list} aria-label={t.list}>
          {vm.rows.map((a) => (
            <li key={a.mbid} className={styles.row}>
              <SignalMeter value={a.match} label={all.common.match} />
              <div className={styles.info}>
                {a.subscribed ? (
                  <Link className={styles.name} to={`/artists/${a.mbid}`}>
                    {a.name}
                  </Link>
                ) : (
                  <span className={styles.name}>{a.name}</span>
                )}
                {a.details && <span className={styles.details}>{a.details}</span>}
              </div>
              {a.subscribed ? (
                <span className={styles.subscribed}>
                  <Led state="ok" />
                  {t.subscribed}
                </span>
              ) : (
                <PushButton
                  variant="primary"
                  led={a.subscribing}
                  disabled={a.subscribing}
                  onClick={() => vm.onSubscribe(a.mbid)}
                  aria-label={t.subscribeTo(a.name)}
                >
                  {t.subscribe}
                </PushButton>
              )}
            </li>
          ))}
        </ul>
      )}
    </Panel>
  );
}
