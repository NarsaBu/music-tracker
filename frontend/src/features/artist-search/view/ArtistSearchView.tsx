import { Link } from 'react-router';
import { useI18n } from '@/shared/i18n';
import {
  EngravedInput,
  LcdDisplay,
  Led,
  Panel,
  PushButton,
  SignalMeter,
  ToggleSwitch,
} from '@/shared/ui';
import type { ArtistSearchViewModel } from '../controller/useArtistSearchController';
import styles from './ArtistSearchView.module.css';

const SCALE_MARKS = ['88', '92', '96', '100', '104', '108'];

export function ArtistSearchView(vm: ArtistSearchViewModel) {
  const { t: all } = useI18n();
  const t = all.search;
  return (
    <div className={styles.layout}>
      <Panel
        title={t.title}
        extra={
          <LcdDisplay label={t.state} size="sm">
            {vm.lcdText}
          </LcdDisplay>
        }
      >
        <div className={styles.tuner}>
          <div className={styles.dial}>
            <div className={styles.scale} aria-hidden>
              {SCALE_MARKS.map((m) => (
                <span key={m}>{m}</span>
              ))}
            </div>
            <EngravedInput
              label={t.artist}
              type="search"
              autoComplete="off"
              spellCheck={false}
              placeholder="Radiohead, Björk, AC/DC…"
              value={vm.query}
              onChange={(e) => vm.onQueryChange(e.target.value)}
              hint={t.hint}
            />
          </div>
          <ToggleSwitch
            className={styles.featuring}
            label={all.common.featuring}
            description={all.common.featuringHint}
            checked={vm.includeFeatured}
            onChange={vm.onIncludeFeaturedChange}
          />
        </div>
      </Panel>

      {vm.errorText && (
        <p className={styles.error} role="alert">
          <Led state="warn" />
          {vm.errorText}
          {vm.status === 'error' && <PushButton onClick={vm.onRetry}>{t.retry}</PushButton>}
        </p>
      )}

      {vm.status === 'empty' && <p className={styles.note}>{t.empty}</p>}

      {vm.stations.length > 0 && (
        <ul className={styles.stations} aria-label={t.results} aria-busy={vm.status === 'tuning'}>
          {vm.stations.map((s) => (
            <li key={s.mbid} className={styles.station}>
              <SignalMeter value={s.score} label={all.common.match} />
              <div className={styles.info}>
                <span className={styles.name}>{s.name}</span>
                <span className={styles.details}>
                  {[s.details, s.country, s.years].filter(Boolean).join(' · ')}
                </span>
              </div>
              {s.subscribed ? (
                <span className={styles.subscribed}>
                  <Led state="ok" />
                  {t.subscribed}
                </span>
              ) : (
                <PushButton
                  variant="primary"
                  led={s.subscribing}
                  disabled={s.subscribing}
                  onClick={() => vm.onSubscribe(s.mbid)}
                  aria-label={t.subscribeTo(s.name)}
                >
                  {t.subscribe}
                </PushButton>
              )}
            </li>
          ))}
        </ul>
      )}

      <p className={styles.back}>
        <Link to="/artists">{t.back}</Link>
      </p>
    </div>
  );
}
