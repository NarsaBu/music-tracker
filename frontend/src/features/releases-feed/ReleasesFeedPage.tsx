import { useReleasesFeedController } from './controller/useReleasesFeedController';
import { ReleasesFeedView } from './view/ReleasesFeedView';

export function ReleasesFeedPage() {
  const vm = useReleasesFeedController();
  return <ReleasesFeedView {...vm} />;
}
