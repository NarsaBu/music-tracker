import { artistTypeLabel } from '@/entities/artist/model';
import { useSimilarArtistsQuery } from '@/entities/artist/queries';
import { useSubscribeMutation } from '@/entities/subscription/queries';
import { ApiError } from '@/shared/api';
import { useI18n } from '@/shared/i18n';

export interface SimilarRow {
  mbid: string;
  name: string;
  details: string;
  /** Similarity relative to the closest artist in the list, 0–100. */
  match: number;
  subscribed: boolean;
  subscribing: boolean;
}

/** Similar artists on the artist page: ListenBrainz data, one-click subscribe. */
export function useSimilarArtistsController(mbid: string) {
  const { t } = useI18n();
  const similar = useSimilarArtistsQuery(mbid);
  const subscribe = useSubscribeMutation();

  const items = similar.data ?? [];
  const top = Math.max(1, ...items.map((a) => a.score));
  const rows: SimilarRow[] = items.map((a) => ({
    mbid: a.mbid,
    name: a.name,
    details: [a.disambiguation, artistTypeLabel(a.type, t.common)].filter(Boolean).join(' · '),
    match: Math.round((a.score / top) * 100),
    subscribed: a.subscribed,
    subscribing: subscribe.isPending && subscribe.variables?.mbid === a.mbid,
  }));

  let errorText: string | undefined;
  if (similar.isError) {
    errorText =
      similar.error instanceof ApiError && similar.error.status === 503
        ? t.similar.unavailable
        : t.similar.loadFailed;
  } else if (subscribe.isError) {
    errorText = t.similar.subscribeFailed;
  }

  return {
    rows,
    isLoading: similar.isPending,
    isEmpty: similar.isSuccess && rows.length === 0,
    loadFailed: similar.isError,
    errorText,

    onRetry: () => void similar.refetch(),
    // No override: the global "featuring" setting applies, as on the tuner by default.
    onSubscribe: (artist: string) => subscribe.mutate({ mbid: artist }),
  };
}

export type SimilarArtistsViewModel = ReturnType<typeof useSimilarArtistsController>;
