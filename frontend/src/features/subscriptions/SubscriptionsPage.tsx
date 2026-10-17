import { SyncControl } from '@/features/sync/SyncControl';
import { useSubscriptionsController } from './controller/useSubscriptionsController';
import { SubscriptionsView } from './view/SubscriptionsView';

export function SubscriptionsPage() {
  const vm = useSubscriptionsController();
  return <SubscriptionsView {...vm} header={<SyncControl />} />;
}
