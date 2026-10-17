import { useParams } from 'react-router';
import { useImportReviewController } from './controller/useImportReviewController';
import { ImportReviewView } from './view/ImportReviewView';

export function ImportReviewPage() {
  const { id = '' } = useParams();
  const vm = useImportReviewController(id);
  return <ImportReviewView {...vm} />;
}
