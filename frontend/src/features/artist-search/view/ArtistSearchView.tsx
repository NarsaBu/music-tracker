import { Link } from 'react-router';
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
  return (
    <div className={styles.layout}>
      <Panel
        title="Тюнер · поиск исполнителя"
        extra={
          <LcdDisplay label="Состояние поиска" size="sm">
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
              label="Исполнитель"
              type="search"
              autoComplete="off"
              spellCheck={false}
              placeholder="Radiohead, Björk, AC/DC…"
              value={vm.query}
              onChange={(e) => vm.onQueryChange(e.target.value)}
              hint="Имя или часть имени; поиск по MusicBrainz"
            />
          </div>
          <ToggleSwitch
            className={styles.featuring}
            label="Featuring"
            description="Сообщать и о релизах, где исполнитель — гость. Выключено — как в общих настройках."
            checked={vm.includeFeatured}
            onChange={vm.onIncludeFeaturedChange}
          />
        </div>
      </Panel>

      {vm.errorText && (
        <p className={styles.error} role="alert">
          <Led state="warn" />
          {vm.errorText}
          {vm.status === 'error' && <PushButton onClick={vm.onRetry}>Повторить</PushButton>}
        </p>
      )}

      {vm.status === 'empty' && (
        <p className={styles.note}>
          Ничего не найдено. Проверьте написание или попробуйте другое имя.
        </p>
      )}

      {vm.stations.length > 0 && (
        <ul
          className={styles.stations}
          aria-label="Найденные исполнители"
          aria-busy={vm.status === 'tuning'}
        >
          {vm.stations.map((s) => (
            <li key={s.mbid} className={styles.station}>
              <SignalMeter value={s.score} label="Совпадение" />
              <div className={styles.info}>
                <span className={styles.name}>{s.name}</span>
                <span className={styles.details}>
                  {[s.details, s.country, s.years].filter(Boolean).join(' · ')}
                </span>
              </div>
              {s.subscribed ? (
                <span className={styles.subscribed}>
                  <Led state="ok" />
                  Подписка есть
                </span>
              ) : (
                <PushButton
                  variant="primary"
                  led={s.subscribing}
                  disabled={s.subscribing}
                  onClick={() => vm.onSubscribe(s.mbid)}
                  aria-label={`Подписаться на ${s.name}`}
                >
                  Подписаться
                </PushButton>
              )}
            </li>
          ))}
        </ul>
      )}

      <p className={styles.back}>
        <Link to="/artists">← К картотеке подписок</Link>
      </p>
    </div>
  );
}
