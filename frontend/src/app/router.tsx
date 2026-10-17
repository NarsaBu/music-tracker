import { createBrowserRouter } from 'react-router';
import { ArtistCatalogPage } from '@/features/artist-catalog/ArtistCatalogPage';
import { ArtistSearchPage } from '@/features/artist-search/ArtistSearchPage';
import { SettingsPage } from '@/features/settings/SettingsPage';
import { SubscriptionsPage } from '@/features/subscriptions/SubscriptionsPage';
import { RackLayout } from './layout/RackLayout';
import { PlaceholderPage } from './PlaceholderPage';

export const router = createBrowserRouter([
  {
    element: <RackLayout />,
    children: [
      { index: true, element: <PlaceholderPage title="Новые релизы" stage="STAGE 3" /> },
      { path: 'artists', element: <SubscriptionsPage /> },
      { path: 'artists/add', element: <ArtistSearchPage /> },
      { path: 'artists/:mbid', element: <ArtistCatalogPage /> },
      { path: 'imports', element: <PlaceholderPage title="Импорт" stage="STAGE 4" /> },
      { path: 'settings', element: <SettingsPage /> },
      { path: '*', element: <PlaceholderPage title="Не найдено" stage="404" /> },
    ],
  },
]);
