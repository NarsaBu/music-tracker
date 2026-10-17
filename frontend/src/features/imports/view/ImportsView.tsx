import { Link } from 'react-router';
import { CassetteDropzone, LcdDisplay, Led, Panel, TapeLabel } from '@/shared/ui';
import { ACCEPTED_FILES, type ImportsViewModel } from '../controller/useImportsController';
import styles from './ImportsView.module.css';

export function ImportsView(vm: ImportsViewModel) {
  return (
    <div className={styles.layout}>
      <Panel
        title="Кассетная дека · импорт"
        extra={
          <LcdDisplay label="Состояние деки" size="sm">
            {vm.lcdText}
          </LcdDisplay>
        }
      >
        <div className={styles.deck}>
          <CassetteDropzone
            label="Вставьте кассету"
            tapeLabel={vm.tapeLabel}
            hint="Перетащите файл сюда или нажмите, чтобы выбрать. .txt или .csv, до 1 МБ и 5000 строк."
            accept={ACCEPTED_FILES}
            busy={vm.spinning}
            disabled={vm.uploading}
            onFile={vm.onFile}
          />
          <div className={styles.formats}>
            <h3 className={styles.formatsTitle}>Что записать на кассету</h3>
            <p>
              <TapeLabel>.txt</TapeLabel> — по исполнителю на строку: имя, MBID или ссылка на
              страницу в MusicBrainz. Строки с <code>#</code> пропускаются.
            </p>
            <pre className={styles.sample}>
              {'Radiohead\nSigur Rós\nhttps://musicbrainz.org/artist/…'}
            </pre>
            <p>
              <TapeLabel>.csv</TapeLabel> — первая строка с названиями колонок: <code>name</code>{' '}
              и/или <code>mbid</code>, по желанию <code>include_featured</code> (true/false).
              Разделитель — запятая или точка с запятой.
            </p>
            <pre className={styles.sample}>
              {'name,mbid,include_featured\nBjörk,,true\nNirvana,5b11f4ce-…,'}
            </pre>
            <p className={styles.note}>
              Уверенные совпадения подписываются сами, остальные ждут вашего выбора на экране
              разбора.
            </p>
          </div>
        </div>
        {vm.uploadError && (
          <p className={styles.error} role="alert">
            <Led state="warn" />
            {vm.uploadError}
          </p>
        )}
      </Panel>

      <Panel title="Журнал записей">
        {vm.isError && (
          <p className={styles.error} role="alert">
            <Led state="warn" />
            Не удалось связаться с сервером
          </p>
        )}
        {!vm.isLoading && !vm.isError && vm.rows.length === 0 && (
          <p className={styles.note}>Пока ничего не импортировано.</p>
        )}
        {vm.rows.length > 0 && (
          <ul className={styles.log} aria-label="Импорты">
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
                  aria-label={`Обработано строк: ${r.filename}`}
                />
                <LcdDisplay label="Обработано строк" size="sm">
                  {r.counterText}
                </LcdDisplay>
                <p className={styles.status} aria-live={r.processing ? 'polite' : undefined}>
                  <Led state={r.led} blink={r.processing} />
                  {r.statusText}
                </p>
                <Link className={styles.open} to={`/imports/${r.id}`}>
                  {r.toReview ? `Разобрать (${r.toReview})` : 'Открыть'}
                </Link>
              </li>
            ))}
          </ul>
        )}
      </Panel>
    </div>
  );
}
