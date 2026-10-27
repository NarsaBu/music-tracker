import { useParams } from 'react-router';
import { SimilarArtists } from '@/features/similar-artists/SimilarArtists';
import { SubscriptionFilter } from '@/features/subscription-filter/SubscriptionFilter';
import { useArtistCatalogController } from './controller/useArtistCatalogController';
import { ArtistCatalogView } from './view/ArtistCatalogView';

function ArtistCatalog({ mbid }: { mbid: string }) {
  const vm = useArtistCatalogController(mbid);
  return (
    <ArtistCatalogView
      {...vm}
      filterPanel={<SubscriptionFilter mbid={mbid} />}
      similarPanel={<SimilarArtists mbid={mbid} />}
    />
  );
}

export function ArtistCatalogPage() {
  const { mbid = '' } = useParams();
  // A similar artist links to another artist page: start its state afresh.
  return <ArtistCatalog key={mbid} mbid={mbid} />;
}
