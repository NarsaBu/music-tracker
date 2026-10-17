import { Link } from 'react-router';
import {
  EngravedInput,
  IndexCard,
  LcdDisplay,
  Led,
  Panel,
  PushButton,
  RotarySelector,
  SignalMeter,
  TapeLabel,
  ToggleSwitch,
  VuMeter,
} from '@/shared/ui';
import {
  OTHER,
  TAB_OPTIONS,
  type ImportReviewViewModel,
  type ReviewCard,
} from '../controller/useImportReviewController';
import styles from './ImportReviewView.module.css';

const pad = (n: number) => String(n).padStart(3, '0');

export function ImportReviewView(vm: ImportReviewViewModel) {
  if (vm.notFound) {
    return (
      <Panel title="Разбор импорта">
        <p className={styles.note}>Такого импорта нет.</p>
        <BackLink />
      </Panel>
    );
  }

  return (
    <div className={styles.layout}>
      <Panel
        title={vm.filename ? `Разбор · ${vm.filename}` : 'Разбор импорта'}
        extra={
          vm.processing && (
            <LcdDisplay label="Обработано" size="sm">
              {`${vm.progress}%`}
            </LcdDisplay>
          )
        }
      >
        <div className={styles.summary}>
          {vm.processing && <VuMeter value={vm.progress} label="Прогресс сопоставления" />}
          <dl className={styles.counters}>
            <Counter label="Строк" value={vm.counts.total} />
            <Counter label="Добавлено" value={vm.counts.matched} />
            <Counter label="Ждут разбора" value={vm.counts.toReview} />
            <Counter label="Пропущено" value={vm.counts.skipped} />
            {vm.processing && <Counter label="В очереди" value={vm.counts.pending} />}
          </dl>
          <RotarySelector
            label="Показать"
            options={TAB_OPTIONS}
            value={vm.tab}
            onChange={vm.onTabChange}
          />
        </div>
      </Panel>

      {vm.isError && (
        <p className={styles.message} role="alert">
          <Led state="warn" />
          Не удалось связаться с сервером
        </p>
      )}

      {vm.tab === 'review' && !vm.isLoading && vm.cards.length === 0 && (
        <p className={styles.message}>
          {vm.processing
            ? 'Пока разбирать нечего — строки ещё сопоставляются.'
            : 'Всё разобрано: спорных строк не осталось.'}
        </p>
      )}

      {vm.cards.length > 0 && (
        <ul className={styles.cards} aria-label="Строки для разбора">
          {vm.cards.map((c) => (
            <li key={c.id}>
              <ReviewCardView card={c} vm={vm} />
            </li>
          ))}
        </ul>
      )}

      {vm.tab !== 'review' && !vm.isLoading && vm.rows.length === 0 && (
        <p className={styles.message}>Здесь пусто.</p>
      )}

      {vm.rows.length > 0 && (
        <Panel title="Строки файла">
          <ol className={styles.lines}>
            {vm.rows.map((r) => (
              <li key={r.id} className={styles.line}>
                <span className={styles.lineNo}>{r.lineNo}</span>
                <span className={styles.raw}>{r.rawValue}</span>
                {r.artist ? (
                  <Link to={`/artists/${r.artist.mbid}`}>{r.artist.name}</Link>
                ) : (
                  <span className={styles.lineStatus}>{r.statusText}</span>
                )}
              </li>
            ))}
          </ol>
        </Panel>
      )}

      <BackLink />
    </div>
  );
}

function Counter({ label, value }: { label: string; value: number }) {
  return (
    <div className={styles.counter}>
      <dt>{label}</dt>
      <dd>
        <LcdDisplay size="sm">{pad(value)}</LcdDisplay>
      </dd>
    </div>
  );
}

function BackLink() {
  return (
    <p className={styles.back}>
      <Link to="/imports">← К кассетной деке</Link>
    </p>
  );
}

function ReviewCardView({ card: c, vm }: { card: ReviewCard; vm: ImportReviewViewModel }) {
  const name = `choice-${c.id}`;
  return (
    <IndexCard
      title={c.rawValue}
      tab={`стр. ${c.lineNo}`}
      tiltSeed={c.id}
      extra={
        <span className={styles.mark}>
          <Led state="warn" />
          {c.statusText}
        </span>
      }
      footer={
        <>
          <PushButton
            variant="primary"
            className={styles.stamp}
            disabled={!c.canConfirm}
            led={c.busy}
            onClick={() => vm.onConfirm(c.id)}
            aria-label={`Подписаться: ${c.rawValue}`}
          >
            Подписать
          </PushButton>
          <PushButton
            disabled={c.busy}
            onClick={() => vm.onSkip(c.id)}
            aria-label={`Пропустить: ${c.rawValue}`}
          >
            Пропустить
          </PushButton>
        </>
      }
    >
      {c.error && <p className={styles.cardNote}>Ошибка: {c.error}</p>}
      <fieldset className={styles.choices} disabled={c.busy}>
        <legend className={styles.legend}>
          {c.candidates.length ? 'Кто это?' : 'В MusicBrainz не нашлось — укажите сами'}
        </legend>
        {c.candidates.map((o) => (
          <label key={o.mbid} className={styles.choice}>
            <input
              type="radio"
              name={name}
              value={o.mbid}
              checked={c.choice === o.mbid}
              onChange={() => vm.onChoose(c.id, o.mbid)}
            />
            <span className={styles.choiceText}>
              <span className={styles.choiceName}>{o.name}</span>
              {o.details && <span className={styles.choiceDetails}>{o.details}</span>}
            </span>
            <SignalMeter value={o.score} label="Совпадение" />
          </label>
        ))}
        {c.candidates.length > 0 && (
          <label className={styles.choice}>
            <input
              type="radio"
              name={name}
              value={OTHER}
              checked={c.choice === OTHER}
              onChange={() => vm.onChoose(c.id, OTHER)}
            />
            <span className={styles.choiceText}>
              <span className={styles.choiceName}>Другой исполнитель</span>
            </span>
          </label>
        )}
        {c.choice === OTHER && (
          <EngravedInput
            label="MBID или ссылка"
            autoComplete="off"
            spellCheck={false}
            placeholder="https://musicbrainz.org/artist/…"
            value={c.other}
            error={c.otherError}
            onChange={(e) => vm.onOtherChange(c.id, e.target.value)}
          />
        )}
      </fieldset>
      <ToggleSwitch
        label="Featuring"
        description="Сообщать и о релизах, где исполнитель — гость. Выключено — как в общих настройках."
        checked={c.featured}
        disabled={c.busy}
        onChange={(v) => vm.onFeaturedChange(c.id, v)}
      />
      {c.errorText && (
        <p className={styles.cardError} role="alert">
          {c.errorText}
        </p>
      )}
      {c.busy && <TapeLabel tone="accent">штампую…</TapeLabel>}
    </IndexCard>
  );
}
