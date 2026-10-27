import { Link } from 'react-router';
import { useI18n } from '@/shared/i18n';
import { LcdDisplay, Led, Panel, PushButton, RecordSleeve, TapeLabel, VuMeter } from '@/shared/ui';
import type {
  ArtistLinkModel,
  ReleasePageViewModel,
  ScoreModel,
} from '../controller/useReleasePageController';
import styles from './ReleasePageView.module.css';

function Score({ name, score }: { name: string; score: ScoreModel }) {
  return (
    <div className={styles.score}>
      <span className={styles.source}>{name}</span>
      <VuMeter value={score.percent} label={name} />
      <LcdDisplay label={name} size="sm">
        {score.valueText ?? '—'}
      </LcdDisplay>
      <span className={styles.votes}>{score.votesText}</span>
    </div>
  );
}

function ArtistLinks({
  artists,
  externalHint,
}: {
  artists: ArtistLinkModel[];
  externalHint: string;
}) {
  return (
    <span className={styles.artistLinks}>
      {artists.map((a) =>
        'to' in a ? (
          <Link key={a.mbid} to={a.to}>
            {a.name}
          </Link>
        ) : (
          <a key={a.mbid} href={a.href} target="_blank" rel="noreferrer" title={externalHint}>
            {a.name} ↗
          </a>
        ),
      )}
    </span>
  );
}

export function ReleasePageView(vm: ReleasePageViewModel) {
  const { t: all } = useI18n();
  const t = all.releasePage;

  const back = (
    <p className={styles.back}>
      <button type="button" className={styles.backButton} onClick={vm.onBack}>
        {t.back}
      </button>
    </p>
  );

  if (vm.isLoading || vm.notFound || vm.loadFailed) {
    return (
      <div className={styles.layout}>
        <Panel title={t.title}>
          <p role={vm.loadFailed ? 'alert' : undefined}>
            {vm.isLoading ? t.loading : vm.notFound ? t.notFound : t.loadFailed}
          </p>
        </Panel>
        {back}
      </div>
    );
  }

  return (
    <div className={styles.layout}>
      <Panel title={vm.title}>
        <div className={styles.head}>
          <div className={styles.coverColumn}>
            <RecordSleeve className={styles.cover} title={vm.title} coverUrl={vm.coverUrl} />
            <PushButton
              led={vm.coverSearching}
              disabled={vm.coverSearching}
              onClick={vm.onDownloadCover}
            >
              {t.downloadCover(1200)}
            </PushButton>
            {(vm.coverSearching || vm.coverText) && (
              <p
                className={vm.coverFailed ? styles.error : styles.note}
                role={vm.coverFailed ? 'alert' : 'status'}
              >
                {vm.coverSearching ? t.searchingCover : vm.coverText}
              </p>
            )}
          </div>
          <div className={styles.info}>
            <p className={styles.credit}>{vm.credit}</p>
            <p className={styles.details}>
              {vm.details}
              {vm.announcement && <TapeLabel tone="accent">{all.release.announcement}</TapeLabel>}
            </p>

            <dl className={styles.facts}>
              <dt>{t.released}</dt>
              <dd>{vm.releaseDate}</dd>
              {vm.artists.length > 0 && (
                <>
                  <dt>{t.artists}</dt>
                  <dd>
                    <ArtistLinks artists={vm.artists} externalHint={t.externalArtist} />
                  </dd>
                </>
              )}
              {vm.guests.length > 0 && (
                <>
                  <dt>{t.guests}</dt>
                  <dd>
                    <ArtistLinks artists={vm.guests} externalHint={t.externalArtist} />
                  </dd>
                </>
              )}
              {!vm.infoLoading && !vm.infoError && (
                <>
                  <dt>{t.genres}</dt>
                  <dd className={styles.genres}>
                    {vm.genres.length > 0
                      ? vm.genres.map((g) => <TapeLabel key={g}>{g}</TapeLabel>)
                      : t.noGenres}
                  </dd>
                </>
              )}
            </dl>

            <h3 className={styles.heading}>{t.ratings}</h3>
            <div className={styles.scores}>
              <Score name={t.musicbrainz} score={vm.musicbrainz} />
              <Score name={t.critiquebrainz} score={vm.critiquebrainz} />
            </div>

            {vm.links && (
              <>
                <h3 className={styles.heading}>{t.elsewhere}</h3>
                <p className={styles.links}>
                  <a href={vm.links.albumOfTheYear} target="_blank" rel="noreferrer">
                    {t.albumOfTheYear} ↗
                  </a>
                  <a href={vm.links.rateYourMusic} target="_blank" rel="noreferrer">
                    {t.rateYourMusic} ↗
                  </a>
                  <a href={vm.links.musicBrainz} target="_blank" rel="noreferrer">
                    {t.openMusicBrainz} ↗
                  </a>
                </p>
              </>
            )}
          </div>
        </div>
      </Panel>

      <Panel
        title={t.tracklist}
        extra={<span className={styles.source}>{vm.tracklistSummary || t.tracklistSource}</span>}
      >
        {vm.infoLoading && <p className={styles.note}>{t.infoLoading}</p>}
        {vm.infoError && (
          <p className={styles.error} role="alert">
            <Led state="warn" />
            {vm.infoError}
            <PushButton onClick={vm.onRetryInfo}>{t.retry}</PushButton>
          </p>
        )}
        {vm.noTracks && <p className={styles.note}>{t.noTracks}</p>}
        {vm.media.map((m) => (
          <section key={m.key} className={styles.medium}>
            {m.heading && <h3 className={styles.heading}>{m.heading}</h3>}
            <ol className={styles.tracks}>
              {m.tracks.map((tr) => (
                <li key={tr.key} className={styles.track}>
                  <span className={styles.trackNo}>{tr.number}</span>
                  <span className={styles.trackTitle}>
                    {tr.title}
                    {tr.credit && <span className={styles.trackCredit}>{tr.credit}</span>}
                  </span>
                  <span className={styles.trackLength}>{tr.length}</span>
                </li>
              ))}
            </ol>
          </section>
        ))}
        {vm.editionText && (
          <p className={styles.note}>
            {vm.editionUrl ? (
              <a href={vm.editionUrl} target="_blank" rel="noreferrer">
                {vm.editionText} ↗
              </a>
            ) : (
              vm.editionText
            )}
          </p>
        )}
      </Panel>

      <Panel title={t.reviews} extra={<span className={styles.source}>{t.reviewsSource}</span>}>
        {vm.reviewsLoading && <p className={styles.note}>{t.reviewsLoading}</p>}
        {vm.reviewsError && (
          <p className={styles.error} role="alert">
            <Led state="warn" />
            {vm.reviewsError}
            <PushButton onClick={vm.onRetryReviews}>{t.retry}</PushButton>
          </p>
        )}
        {vm.reviewsEmpty && <p className={styles.note}>{t.reviewsEmpty}</p>}

        {vm.reviews.length > 0 && (
          <ul className={styles.reviews}>
            {vm.reviews.map((r) => (
              <li key={r.id} className={styles.review}>
                <p className={styles.reviewHead}>
                  <span className={styles.author}>{r.author}</span>
                  {r.stars !== null && (
                    <span className={styles.stars} role="img" aria-label={t.stars(r.stars)}>
                      {'★'.repeat(r.stars)}
                      {'☆'.repeat(5 - r.stars)}
                    </span>
                  )}
                  {r.date && <span className={styles.date}>{r.date}</span>}
                </p>
                <p className={styles.text}>{r.text}</p>
                <p className={styles.reviewFoot}>
                  <a href={r.url} target="_blank" rel="noreferrer">
                    {t.readFull} ↗
                  </a>
                  {r.license && <span className={styles.license}>{r.license}</span>}
                </p>
              </li>
            ))}
          </ul>
        )}

        {vm.allReviewsUrl && (
          <p className={styles.note}>
            <a href={vm.allReviewsUrl} target="_blank" rel="noreferrer">
              {t.allReviews(vm.reviewsTotal)} ↗
            </a>
          </p>
        )}
      </Panel>

      {back}
    </div>
  );
}
