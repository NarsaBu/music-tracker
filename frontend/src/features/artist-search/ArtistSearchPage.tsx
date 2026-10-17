import { useArtistSearchController } from './controller/useArtistSearchController';
import { ArtistSearchView } from './view/ArtistSearchView';

export function ArtistSearchPage() {
  const vm = useArtistSearchController();
  return <ArtistSearchView {...vm} />;
}
