import { useState } from 'react';
import { formatLifeSpan } from '@/entities/artist/model';
import {
  isResolvable,
  progressOf,
  toReviewOf,
  type ImportItem,
  type ImportItemStatus,
} from '@/entities/import/model';
import {
  IMPORT_POLL_MS,
  useImportItemsQuery,
  useImportQuery,
  useResolveItemMutation,
} from '@/entities/import/queries';
import { ApiError } from '@/shared/api';

export type ReviewTab = 'review' | 'matched' | 'skipped' | 'all';

const TAB_STATUSES: Record<ReviewTab, readonly ImportItemStatus[]> = {
  review: ['needs_review', 'not_found', 'failed'],
  matched: ['matched'],
  skipped: ['skipped'],
  all: [],
};

export const TAB_OPTIONS: ReadonlyArray<{ value: ReviewTab; label: string }> = [
  { value: 'review', label: 'Разбор' },
  { value: 'matched', label: 'Добавлены' },
  { value: 'skipped', label: 'Пропущены' },
  { value: 'all', label: 'Все' },
];

export const STATUS_TEXT: Record<ImportItemStatus, string> = {
  pending: 'в очереди',
  matched: 'подписка оформлена',
  needs_review: 'несколько тёзок',
  not_found: 'не найден',
  skipped: 'пропущен',
  failed: 'ошибка',
};

/** Value of the "someone else" option of a card. */
export const OTHER = 'other';

interface Draft {
  /** Candidate MBID or OTHER. */
  choice?: string;
  other: string;
  featured?: boolean;
  otherError?: string;
}

export interface CandidateOption {
  mbid: string;
  name: string;
  details: string;
  score: number;
}

export interface ReviewCard {
  id: string;
  lineNo: number;
  rawValue: string;
  statusText: string;
  /** Why the worker gave up (failed lines). */
  error: string | null;
  candidates: CandidateOption[];
  choice: string | undefined;
  other: string;
  otherError: string | undefined;
  featured: boolean;
  canConfirm: boolean;
  busy: boolean;
  errorText: string | undefined;
}

export interface LineRow {
  id: string;
  lineNo: number;
  rawValue: string;
  statusText: string;
  artist: { mbid: string; name: string } | null;
}

function resolveErrorText(error: unknown): string {
  if (error instanceof ApiError) {
    if (error.status === 409) return 'Эту строку уже разобрали';
    if (error.status === 503) return 'MusicBrainz не отвечает, попробуйте через минуту';
    if (error.status === 422)
      return `Не принято: ${error.fields?.[0]?.message ?? error.detail ?? ''}`;
  }
  return 'Не удалось сохранить выбор';
}

const featuredOf = (item: ImportItem) => item.includeFeatured ?? false;

/** Import review screen ("index cards with marks", US-3): settle ambiguous lines. */
export function useImportReviewController(id: string) {
  const [tab, setTab] = useState<ReviewTab>('review');
  const [drafts, setDrafts] = useState<Record<string, Draft>>({});

  const job = useImportQuery(id);
  const processing = !!job.data && !job.data.finished;
  const items = useImportItemsQuery(id, TAB_STATUSES[tab], {
    pollMs: processing ? IMPORT_POLL_MS : false,
  });
  const resolve = useResolveItemMutation(id);

  const draftOf = (itemId: string): Draft => drafts[itemId] ?? { other: '' };
  const patchDraft = (itemId: string, patch: Partial<Draft>) =>
    setDrafts((d) => ({ ...d, [itemId]: { ...(d[itemId] ?? { other: '' }), ...patch } }));
  const busyId = resolve.isPending ? resolve.variables?.itemId : undefined;
  const failedId = resolve.isError ? resolve.variables?.itemId : undefined;

  const data = items.data ?? [];
  const cards: ReviewCard[] =
    tab === 'review'
      ? data
          .filter((it) => isResolvable(it.status))
          .map((it) => {
            const d = draftOf(it.id);
            const choice = d.choice ?? (it.candidates.length ? undefined : OTHER);
            return {
              id: it.id,
              lineNo: it.lineNo,
              rawValue: it.rawValue,
              statusText: STATUS_TEXT[it.status],
              error: it.error,
              candidates: it.candidates.map((c) => ({
                mbid: c.mbid,
                name: c.name,
                details: [c.disambiguation, c.type, c.country, formatLifeSpan(c.lifeSpan)]
                  .filter(Boolean)
                  .join(' · '),
                score: c.score,
              })),
              choice,
              other: d.other,
              otherError: d.otherError,
              featured: d.featured ?? featuredOf(it),
              canConfirm: !!choice && busyId !== it.id,
              busy: busyId === it.id,
              errorText: failedId === it.id ? resolveErrorText(resolve.error) : undefined,
            };
          })
      : [];
  const rows: LineRow[] =
    tab === 'review'
      ? []
      : data.map((it) => ({
          id: it.id,
          lineNo: it.lineNo,
          rawValue: it.rawValue,
          statusText: STATUS_TEXT[it.status],
          artist: it.artist,
        }));

  const j = job.data;
  const notFound = job.error instanceof ApiError && job.error.status === 404;
  return {
    filename: j?.filename ?? '',
    notFound,
    isError: (job.isError && !notFound) || items.isError,
    isLoading: job.isPending || items.isPending,
    processing,
    progress: j ? progressOf(j) : 0,
    counts: {
      total: j?.total ?? 0,
      matched: j?.counts.matched ?? 0,
      toReview: j ? toReviewOf(j) : 0,
      skipped: j?.counts.skipped ?? 0,
      pending: j?.counts.pending ?? 0,
    },
    tab,
    cards,
    rows,

    onTabChange: setTab,
    onChoose: (itemId: string, choice: string) =>
      patchDraft(itemId, { choice, otherError: undefined }),
    onOtherChange: (itemId: string, other: string) =>
      patchDraft(itemId, { other, choice: OTHER, otherError: undefined }),
    onFeaturedChange: (itemId: string, featured: boolean) => patchDraft(itemId, { featured }),
    onConfirm: (itemId: string) => {
      const item = data.find((it) => it.id === itemId);
      const card = cards.find((c) => c.id === itemId);
      if (!item || !card?.choice) return;
      const mbid = card.choice === OTHER ? card.other.trim() : card.choice;
      if (!mbid) {
        patchDraft(itemId, { otherError: 'Вставьте MBID или ссылку на исполнителя в MusicBrainz' });
        return;
      }
      // Only a change is sent: otherwise the CSV value (or the global setting) stays.
      const includeFeatured = card.featured !== featuredOf(item) ? card.featured : undefined;
      resolve.mutate(
        { itemId, req: { mbid, includeFeatured } },
        {
          onSuccess: () =>
            setDrafts((d) => {
              const rest = { ...d };
              delete rest[itemId];
              return rest;
            }),
        },
      );
    },
    onSkip: (itemId: string) => resolve.mutate({ itemId, req: { skip: true } }),
  };
}

export type ImportReviewViewModel = ReturnType<typeof useImportReviewController>;
