import { createBrowserRouter } from 'react-router';
import { ArtistCatalogPage } from '@/features/artist-catalog/ArtistCatalogPage';
import { ArtistSearchPage } from '@/features/artist-search/ArtistSearchPage';
import { ImportReviewPage } from '@/features/import-review/ImportReviewPage';
import { ImportsPage } from '@/features/imports/ImportsPage';
import { ReleasesFeedPage } from '@/features/releases-feed/ReleasesFeedPage';
import { SettingsPage } from '@/features/settings/SettingsPage';
import { SubscriptionsPage } from '@/features/subscriptions/SubscriptionsPage';
import { RackLayout } from './layout/RackLayout';
import { PlaceholderPage } from './PlaceholderPage';

export const router = createBrowserRouter([
  {
    element: <RackLayout />,
    children: [
      { index: true, element: <ReleasesFeedPage /> },
      { path: 'artists', element: <SubscriptionsPage /> },
      { path: 'artists/add', element: <ArtistSearchPage /> },
      { path: 'artists/:mbid', element: <ArtistCatalogPage /> },
      { path: 'imports', element: <ImportsPage /> },
      { path: 'imports/:id', element: <ImportReviewPage /> },
      { path: 'settings', element: <SettingsPage /> },
      { path: '*', element: <PlaceholderPage stage="404" /> },
    ],
  },
]);
