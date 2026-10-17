import { keepPreviousData, useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { artistKeys } from '@/entities/artist/queries';
import type { ArtistCandidate } from '@/entities/artist/model';
import { httpClient } from '@/shared/api';
import { createSubscriptionApi } from './api';
import type { SubscribeRequest, SubscriptionListParams } from './model';

const subscriptionApi = createSubscriptionApi(httpClient);

export const subscriptionKeys = {
  all: ['subscriptions'] as const,
  list: (params: SubscriptionListParams) => [...subscriptionKeys.all, 'list', params] as const,
};

export function useSubscriptionsQuery(params: SubscriptionListParams) {
  return useQuery({
    queryKey: subscriptionKeys.list(params),
    queryFn: () => subscriptionApi.list(params),
    placeholderData: keepPreviousData,
  });
}

/**
 * Marks the artist in every cached search result instead of refetching:
 * a new search would cost a MusicBrainz request (1 rps).
 */
function useMarkSubscribed() {
  const queryClient = useQueryClient();
  return (mbid: string, subscribed: boolean) =>
    queryClient.setQueriesData<ArtistCandidate[]>({ queryKey: artistKeys.all }, (items) =>
      items?.map((a) => (a.mbid === mbid ? { ...a, subscribed } : a)),
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
