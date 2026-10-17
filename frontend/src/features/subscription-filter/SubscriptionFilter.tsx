import { useSubscriptionFilterController } from './controller/useSubscriptionFilterController';
import { SubscriptionFilterView } from './view/SubscriptionFilterView';

/** Self-contained widget for the artist page. */
export function SubscriptionFilter({ mbid }: { mbid: string }) {
  const vm = useSubscriptionFilterController(mbid);
  return <SubscriptionFilterView {...vm} />;
}
