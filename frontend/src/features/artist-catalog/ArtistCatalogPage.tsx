import { useParams } from 'react-router';
import { SubscriptionFilter } from '@/features/subscription-filter/SubscriptionFilter';
import { useArtistCatalogController } from './controller/useArtistCatalogController';
import { ArtistCatalogView } from './view/ArtistCatalogView';

export function ArtistCatalogPage() {
  const { mbid = '' } = useParams();
  const vm = useArtistCatalogController(mbid);
  return <ArtistCatalogView {...vm} filterPanel={<SubscriptionFilter mbid={mbid} />} />;
}
