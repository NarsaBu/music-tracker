import { Link } from 'react-router';
import { useI18n } from '@/shared/i18n';
import { CassetteDropzone, LcdDisplay, Led, Panel, TapeLabel } from '@/shared/ui';
import { ACCEPTED_FILES, type ImportsViewModel } from '../controller/useImportsController';
import styles from './ImportsView.module.css';

export function ImportsView(vm: ImportsViewModel) {
  const { t: all } = useI18n();
  const t = all.imports;
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
        <div className={styles.deck}>
          <CassetteDropzone
            label={t.insert}
            tapeLabel={vm.tapeLabel ?? t.tape}
            hint={t.hint}
            accept={ACCEPTED_FILES}
            busy={vm.spinning}
            disabled={vm.uploading}
            onFile={vm.onFile}
          />
          <div className={styles.formats}>
            <h3 className={styles.formatsTitle}>{t.formatsTitle}</h3>
            <p>
              <TapeLabel>.txt</TapeLabel>
              {t.txtBefore}
              <code>#</code>
              {t.txtAfter}
            </p>
            <pre className={styles.sample}>
              {'Radiohead\nSigur Rós\nhttps://musicbrainz.org/artist/…'}
            </pre>
            <p>
              <TapeLabel>.csv</TapeLabel>
              {t.csvBefore}
              <code>name</code>
              {t.csvAnd}
              <code>mbid</code>
              {t.csvOptional}
              <code>include_featured</code>
              {t.csvAfter}
            </p>
            <pre className={styles.sample}>
              {'name,mbid,include_featured\nBjörk,,true\nNirvana,5b11f4ce-…,'}
            </pre>
            <p className={styles.note}>{t.autoNote}</p>
          </div>
        </div>
        {vm.uploadError && (
          <p className={styles.error} role="alert">
            <Led state="warn" />
            {vm.uploadError}
          </p>
        )}
      </Panel>

      <Panel title={t.log}>
        {vm.isError && (
          <p className={styles.error} role="alert">
            <Led state="warn" />
            {all.common.serverUnreachable}
          </p>
        )}
        {!vm.isLoading && !vm.isError && vm.rows.length === 0 && (
          <p className={styles.note}>{t.none}</p>
        )}
        {vm.rows.length > 0 && (
          <ul className={styles.log} aria-label={t.list}>
            {vm.rows.map((r) => (
              <li key={r.id} className={styles.row}>
                <div className={styles.file}>
                  <TapeLabel>{r.filename}</TapeLabel>
                  <span className={styles.date}>{r.createdText}</span>
                </div>
                <progress
                  className={styles.progress}
                  max={100}
                  value={r.progress}
                  aria-label={t.processedOf(r.filename)}
                />
                <LcdDisplay label={t.processed} size="sm">
                  {r.counterText}
                </LcdDisplay>
                <p className={styles.status} aria-live={r.processing ? 'polite' : undefined}>
                  <Led state={r.led} blink={r.processing} />
                  {r.statusText}
                </p>
                <Link className={styles.open} to={`/imports/${r.id}`}>
                  {r.toReview ? t.review(r.toReview) : t.open}
                </Link>
              </li>
            ))}
          </ul>
        )}
      </Panel>
    </div>
  );
}
