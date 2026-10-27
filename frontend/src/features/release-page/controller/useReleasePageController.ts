import { useNavigate } from 'react-router';
import {
  artistLink,
  coverDownloadPath,
  formatLength,
  formatPartialDate,
  ratingSearchLinks,
  type CreditedArtist,
  type Rating,
} from '@/entities/release/model';
import {
  useCoverInfoMutation,
  useReleaseInfoQuery,
  useReleaseQuery,
  useReviewsQuery,
} from '@/entities/release/queries';
import { ApiError } from '@/shared/api';
import { useI18n } from '@/shared/i18n';
import { startDownload } from '@/shared/lib/download';

export interface ScoreModel {
  /** "4,6 / 5"; null when there are no votes. */
  valueText: string | null;
  votesText: string;
  /** 0–100, for a meter. */
  percent: number;
}

export interface ReviewRow {
  id: string;
  author: string;
  /** 1–5; null when the author gave none. */
  stars: number | null;
  date: string;
  text: string;
  url: string;
  license: string;
}

export type ArtistLinkModel = { mbid: string; name: string; external: boolean } & (
  { to: string } | { href: string }
);

export interface TrackRow {
  key: string;
  number: string;
  title: string;
  length: string;
  /** The track's own credit when it differs from the release, e.g. "A feat. B". */
  credit: string | null;
}

export interface MediumModel {
  key: string;
  /** "Диск 2 · Vinyl"; null for a single medium without a title. */
  heading: string | null;
  tracks: TrackRow[];
}

/**
 * Release page: what the release is (genres, date, artists, tracklist, guests),
 * MusicBrainz rating, CritiqueBrainz reviews, links to rating sites.
 */
export function useReleasePageController(mbid: string) {
  const { t: all } = useI18n();
  const t = all.releasePage;
  const navigate = useNavigate();
  const release = useReleaseQuery(mbid);
  const reviews = useReviewsQuery(mbid);
  const info = useReleaseInfoQuery(mbid);
  const cover = useCoverInfoMutation();

  const number = new Intl.NumberFormat(all.locale, {
    minimumFractionDigits: 1,
    maximumFractionDigits: 1,
  });
  const score = (r: Rating | null | undefined, votes: (n: number) => string): ScoreModel =>
    r
      ? {
          valueText: t.outOf(number.format(r.value)),
          votesText: votes(r.votes),
          percent: Math.round((r.value / 5) * 100),
        }
      : { valueText: null, votesText: t.noVotes, percent: 0 };
  const dateFormat = new Intl.DateTimeFormat(all.locale, { dateStyle: 'medium' });

  const r = release.data;
  const notFound = release.error instanceof ApiError && release.error.status === 404;
  const links = r ? ratingSearchLinks(r.artistCredit, r.title) : null;

  const reviewRows: ReviewRow[] = (reviews.data?.items ?? []).map((v) => ({
    id: v.id,
    author: v.author,
    stars: v.rating,
    date: v.publishedAt ? dateFormat.format(new Date(v.publishedAt)) : '',
    text: v.text,
    url: v.url,
    license: v.license,
  }));

  const toLink = (a: CreditedArtist): ArtistLinkModel => ({
    mbid: a.mbid,
    name: a.name,
    external: !a.subscribed,
    ...artistLink(a),
  });

  const media = info.data?.media ?? [];
  const tracks = media.flatMap((m) => m.tracks);
  const totalMs = tracks.reduce((sum, tr) => sum + (tr.lengthMs ?? 0), 0);
  const mediaModels: MediumModel[] = media.map((m) => ({
    key: String(m.position),
    heading:
      media.length > 1 || m.title
        ? [t.disc(m.position), m.title, m.format].filter(Boolean).join(' · ')
        : null,
    tracks: m.tracks.map((tr, i) => ({
      key: `${m.position}-${i}`,
      number: tr.number,
      title: tr.title,
      length: tr.lengthMs ? formatLength(tr.lengthMs) : '',
      credit: r && tr.artistCredit !== r.artistCredit ? tr.artistCredit : null,
    })),
  }));
  const edition = info.data?.edition;

  let infoError: string | undefined;
  if (info.isError) {
    infoError =
      info.error instanceof ApiError && info.error.status === 503
        ? t.infoUnavailable
        : t.infoFailed;
  }

  const megabytes = new Intl.NumberFormat(all.locale, { maximumFractionDigits: 1 });
  let coverText: string | undefined;
  let coverFailed = false;
  if (cover.data?.available) {
    const c = cover.data;
    coverText = t.coverStarted(
      [
        `${c.width}×${c.height}`,
        c.format.replace('image/', '').toUpperCase(),
        c.sizeBytes ? `${megabytes.format(c.sizeBytes / 1024 / 1024)} MB` : null,
      ]
        .filter(Boolean)
        .join(' · '),
    );
  } else if (cover.data) {
    const { largest, minSize } = cover.data;
    coverText = largest
      ? t.coverTooSmall(minSize, `${largest.width}×${largest.height}`)
      : t.noCover;
  } else if (cover.isError) {
    coverFailed = true;
    coverText =
      cover.error instanceof ApiError && cover.error.status === 503
        ? t.coverUnavailable
        : t.coverFailed;
  }

  let reviewsError: string | undefined;
  if (reviews.isError) {
    reviewsError =
      reviews.error instanceof ApiError && reviews.error.status === 503
        ? t.reviewsUnavailable
        : t.reviewsFailed;
  }

  return {
    isLoading: release.isPending,
    notFound,
    loadFailed: release.isError && !notFound,
    title: r?.title ?? '',
    credit: r?.artistCredit ?? '',
    details: r
      ? [
          r.primaryType ? all.release.primaryTypes[r.primaryType] : null,
          ...r.secondaryTypes.map((s) => all.release.secondaryTypes[s] ?? s),
        ]
          .filter(Boolean)
          .join(' · ')
      : '',
    releaseDate: r ? formatPartialDate(r.firstReleaseDate, all.common) : '',
    /** Linked artists once MusicBrainz answered; until then the plain credit is shown. */
    artists: (info.data?.artists ?? []).map(toLink),
    guests: (info.data?.guests ?? []).map(toLink),
    genres: info.data?.genres ?? [],

    infoLoading: info.isPending,
    infoError,
    media: mediaModels,
    noTracks: info.isSuccess && tracks.length === 0,
    tracklistSummary:
      tracks.length > 0 ? t.totalLength(tracks.length, totalMs ? formatLength(totalMs) : '—') : '',
    editionText: edition
      ? t.edition(
          [edition.title, edition.country, formatPartialDate(edition.date, all.common)]
            .filter(Boolean)
            .join(' · '),
        )
      : '',
    editionUrl: edition?.musicbrainzUrl,
    announcement: r?.isAnnouncement ?? false,
    coverUrl: r?.coverUrl ?? '',
    musicbrainz: score(r?.rating, t.votes),
    critiquebrainz: score(reviews.data?.average, t.rated),
    links: links && r ? { ...links, musicBrainz: r.musicbrainzUrl } : null,

    reviews: reviewRows,
    reviewsLoading: reviews.isPending,
    reviewsEmpty: reviews.isSuccess && reviewRows.length === 0,
    reviewsError,
    reviewsTotal: reviews.data?.total ?? 0,
    /** Shown when CritiqueBrainz has more than the page holds. */
    allReviewsUrl:
      reviews.data && reviews.data.total > reviewRows.length ? reviews.data.url : undefined,

    onRetryReviews: () => void reviews.refetch(),
    onRetryInfo: () => void info.refetch(),

    coverSearching: cover.isPending,
    coverText,
    coverFailed,
    onDownloadCover: () =>
      cover.mutate(mbid, {
        onSuccess: (c) => {
          if (c.available) startDownload(coverDownloadPath(mbid), c.filename);
        },
      }),
    onBack: () => void navigate(-1),
  };
}

export type ReleasePageViewModel = ReturnType<typeof useReleasePageController>;
