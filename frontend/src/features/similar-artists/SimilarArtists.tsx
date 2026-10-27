import { useSimilarArtistsController } from './controller/useSimilarArtistsController';
import { SimilarArtistsView } from './view/SimilarArtistsView';

/** Self-contained widget for the artist page. */
export function SimilarArtists({ mbid }: { mbid: string }) {
  const vm = useSimilarArtistsController(mbid);
  return <SimilarArtistsView {...vm} />;
}
