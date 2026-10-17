import { useParams } from 'react-router';
import { useArtistCatalogController } from './controller/useArtistCatalogController';
import { ArtistCatalogView } from './view/ArtistCatalogView';

export function ArtistCatalogPage() {
  const { mbid = '' } = useParams();
  const vm = useArtistCatalogController(mbid);
  return <ArtistCatalogView {...vm} />;
}
