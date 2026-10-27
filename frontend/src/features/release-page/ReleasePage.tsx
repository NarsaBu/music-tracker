import { useParams } from 'react-router';
import { useReleasePageController } from './controller/useReleasePageController';
import { ReleasePageView } from './view/ReleasePageView';

function Release({ mbid }: { mbid: string }) {
  const vm = useReleasePageController(mbid);
  return <ReleasePageView {...vm} />;
}

export function ReleasePage() {
  const { mbid = '' } = useParams();
  return <Release key={mbid} mbid={mbid} />;
}
