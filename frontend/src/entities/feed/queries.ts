import { useInfiniteQuery } from '@tanstack/react-query';
import { httpClient } from '@/shared/api';
import { createFeedApi } from './api';
import type { FeedFilter } from './model';

const feedApi = createFeedApi(httpClient);

export const FEED_PAGE_SIZE = 48;

export const feedKeys = {
  all: ['feed'] as const,
  list: (filter: FeedFilter) => [...feedKeys.all, filter] as const,
};

export function useFeedQuery(filter: FeedFilter) {
  return useInfiniteQuery({
    queryKey: feedKeys.list(filter),
    queryFn: ({ pageParam }) => feedApi.page(filter, pageParam, FEED_PAGE_SIZE),
    initialPageParam: null as string | null,
    getNextPageParam: (last) => last.nextCursor,
  });
}
