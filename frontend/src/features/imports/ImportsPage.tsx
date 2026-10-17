import { useImportsController } from './controller/useImportsController';
import { ImportsView } from './view/ImportsView';

export function ImportsPage() {
  const vm = useImportsController();
  return <ImportsView {...vm} />;
}
