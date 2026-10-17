import { Outlet } from 'react-router';
import { useRackLayoutController } from './controller/useRackLayoutController';
import { RackLayoutView } from './view/RackLayoutView';

export function RackLayout() {
  const vm = useRackLayoutController();
  return (
    <RackLayoutView {...vm}>
      <Outlet />
    </RackLayoutView>
  );
}
