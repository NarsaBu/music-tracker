import { createBrowserRouter } from 'react-router';
import { SettingsPage } from '@/features/settings/SettingsPage';
import { RackLayout } from './layout/RackLayout';
import { PlaceholderPage } from './PlaceholderPage';

export const router = createBrowserRouter([
  {
    element: <RackLayout />,
    children: [
      { index: true, element: <PlaceholderPage title="Новые релизы" stage="STAGE 3" /> },
      { path: 'artists', element: <PlaceholderPage title="Исполнители" stage="STAGE 1" /> },
      { path: 'imports', element: <PlaceholderPage title="Импорт" stage="STAGE 4" /> },
      { path: 'settings', element: <SettingsPage /> },
      { path: '*', element: <PlaceholderPage title="Не найдено" stage="404" /> },
    ],
  },
]);
