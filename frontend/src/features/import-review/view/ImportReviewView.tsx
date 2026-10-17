import { Link } from 'react-router';
import { useI18n } from '@/shared/i18n';
import { lcdCount } from '@/shared/lib/format';
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
  type ImportReviewViewModel,
  type ReviewCard,
} from '../controller/useImportReviewController';
import styles from './ImportReviewView.module.css';

export function ImportReviewView(vm: ImportReviewViewModel) {
  const { t: all } = useI18n();
  const t = all.review;
  if (vm.notFound) {
    return (
      <Panel title={t.title}>
        <p className={styles.note}>{t.notFound}</p>
        <BackLink />
      </Panel>
    );
  }

  return (
    <div className={styles.layout}>
      <Panel
        title={vm.filename ? t.titleOf(vm.filename) : t.title}
        extra={
          vm.processing && (
            <LcdDisplay label={t.processed} size="sm">
              {`${vm.progress}%`}
            </LcdDisplay>
          )
        }
      >
        <div className={styles.summary}>
          {vm.processing && <VuMeter value={vm.progress} label={t.progress} />}
          <dl className={styles.counters}>
            <Counter label={t.lines} value={vm.counts.total} />
            <Counter label={t.added} value={vm.counts.matched} />
            <Counter label={t.toReview} value={vm.counts.toReview} />
            <Counter label={t.skipped} value={vm.counts.skipped} />
            {vm.processing && <Counter label={t.queued} value={vm.counts.pending} />}
          </dl>
          <RotarySelector
            label={t.show}
            options={vm.tabOptions}
            value={vm.tab}
            onChange={vm.onTabChange}
          />
        </div>
      </Panel>

      {vm.isError && (
        <p className={styles.message} role="alert">
          <Led state="warn" />
          {all.common.serverUnreachable}
        </p>
      )}

      {vm.tab === 'review' && !vm.isLoading && vm.cards.length === 0 && (
        <p className={styles.message}>{vm.processing ? t.stillMatching : t.allDone}</p>
      )}

      {vm.cards.length > 0 && (
        <ul className={styles.cards} aria-label={t.cards}>
          {vm.cards.map((c) => (
            <li key={c.id}>
              <ReviewCardView card={c} vm={vm} />
            </li>
          ))}
        </ul>
      )}

      {vm.tab !== 'review' && !vm.isLoading && vm.rows.length === 0 && (
        <p className={styles.message}>{t.empty}</p>
      )}

      {vm.rows.length > 0 && (
        <Panel title={t.fileLines}>
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
        <LcdDisplay size="sm">{lcdCount(value)}</LcdDisplay>
      </dd>
    </div>
  );
}

function BackLink() {
  const { t } = useI18n();
  return (
    <p className={styles.back}>
      <Link to="/imports">{t.review.back}</Link>
    </p>
  );
}

function ReviewCardView({ card: c, vm }: { card: ReviewCard; vm: ImportReviewViewModel }) {
  const { t: all } = useI18n();
  const t = all.review;
  const name = `choice-${c.id}`;
  return (
    <IndexCard
      title={c.rawValue}
      tab={t.line(c.lineNo)}
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
            aria-label={t.confirmOf(c.rawValue)}
          >
            {t.confirm}
          </PushButton>
          <PushButton
            disabled={c.busy}
            onClick={() => vm.onSkip(c.id)}
            aria-label={t.skipOf(c.rawValue)}
          >
            {t.skip}
          </PushButton>
        </>
      }
    >
      {c.error && <p className={styles.cardNote}>{t.error(c.error)}</p>}
      <fieldset className={styles.choices} disabled={c.busy}>
        <legend className={styles.legend}>
          {c.candidates.length ? t.whoIsIt : t.notInMusicBrainz}
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
            <SignalMeter value={o.score} label={all.common.match} />
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
              <span className={styles.choiceName}>{t.other}</span>
            </span>
          </label>
        )}
        {c.choice === OTHER && (
          <EngravedInput
            label={t.mbidOrLink}
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
        label={all.common.featuring}
        description={all.common.featuringHint}
        checked={c.featured}
        disabled={c.busy}
        onChange={(v) => vm.onFeaturedChange(c.id, v)}
      />
      {c.errorText && (
        <p className={styles.cardError} role="alert">
          {c.errorText}
        </p>
      )}
      {c.busy && <TapeLabel tone="accent">{t.stamping}</TapeLabel>}
    </IndexCard>
  );
}
