import { keepPreviousData, useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { artistKeys } from '@/entities/artist/queries';
import type { ArtistCandidate, SimilarArtist } from '@/entities/artist/model';
import { httpClient } from '@/shared/api';
import { createSubscriptionApi } from './api';
import type { FilterPatch, SubscribeRequest, SubscriptionListParams } from './model';

const subscriptionApi = createSubscriptionApi(httpClient);

export const subscriptionKeys = {
  all: ['subscriptions'] as const,
  list: (params: SubscriptionListParams) => [...subscriptionKeys.all, 'list', params] as const,
  one: (mbid: string) => [...subscriptionKeys.all, 'one', mbid] as const,
};

export interface SubscriptionQueryOptions {
  /** Poll interval in ms, e.g. while waiting for a requested check; false = no polling. */
  pollMs?: number | false;
}

export function useSubscriptionQuery(
  mbid: string,
  { pollMs = false }: SubscriptionQueryOptions = {},
) {
  return useQuery({
    queryKey: subscriptionKeys.one(mbid),
    queryFn: () => subscriptionApi.get(mbid),
    refetchInterval: pollMs,
  });
}

export function useUpdateFilterMutation(mbid: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (patch: FilterPatch) => subscriptionApi.updateFilter(mbid, patch),
    onSuccess: (sub) => {
      queryClient.setQueryData(subscriptionKeys.one(mbid), sub);
      return queryClient.invalidateQueries({ queryKey: [...subscriptionKeys.all, 'list'] });
    },
  });
}

export function useRequestSyncMutation() {
  return useMutation({ mutationFn: (mbid: string) => subscriptionApi.requestSync(mbid) });
}

export function useSubscriptionsQuery(params: SubscriptionListParams) {
  return useQuery({
    queryKey: subscriptionKeys.list(params),
    queryFn: () => subscriptionApi.list(params),
    placeholderData: keepPreviousData,
  });
}

/**
 * Marks the artist in every cached artist list (search results, similar artists)
 * instead of refetching: a new search would cost a MusicBrainz request (1 rps).
 */
function useMarkSubscribed() {
  const queryClient = useQueryClient();
  return (mbid: string, subscribed: boolean) =>
    queryClient.setQueriesData<Array<ArtistCandidate | SimilarArtist>>(
      { queryKey: artistKeys.all },
      (items) => items?.map((a) => (a.mbid === mbid ? { ...a, subscribed } : a)),
    );
}

export function useSubscribeMutation() {
  const queryClient = useQueryClient();
  const markSubscribed = useMarkSubscribed();
  return useMutation({
    mutationFn: (req: SubscribeRequest) => subscriptionApi.subscribe(req),
    onSuccess: (_sub, req) => {
      markSubscribed(req.mbid, true);
      return queryClient.invalidateQueries({ queryKey: subscriptionKeys.all });
    },
  });
}

export function useUnsubscribeMutation() {
  const queryClient = useQueryClient();
  const markSubscribed = useMarkSubscribed();
  return useMutation({
    mutationFn: (mbid: string) => subscriptionApi.unsubscribe(mbid),
    onSuccess: (_void, mbid) => {
      markSubscribed(mbid, false);
      return queryClient.invalidateQueries({ queryKey: subscriptionKeys.all });
    },
  });
}
