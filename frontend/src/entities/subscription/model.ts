import type { Artist } from '@/entities/artist/model';
import type { PrimaryType, SecondaryType } from '@/entities/release/model';

/** A filter field of a subscription: its effective value and whether it differs from settings. */
export interface Overridable<T> {
  value: T;
  overridden: boolean;
}

export interface SubscriptionFilter {
  primaryTypes: Overridable<PrimaryType[]>;
  excludedSecondaryTypes: Overridable<string[]>;
  includeFeatured: Overridable<boolean>;
}

export interface Subscription {
  artist: Artist;
  filter: SubscriptionFilter;
  /** ISO timestamp; null until the first (baseline) synchronization. */
  lastSyncedAt: string | null;
  baselineAt: string;
  createdAt: string;
}

export interface SubscriptionPage {
  items: Subscription[];
  total: number;
}

export interface SubscriptionListParams {
  search: string;
  limit: number;
  offset: number;
}

/** Overrides are optional: an absent field means "use the global settings". */
export interface SubscribeRequest {
  mbid: string;
  includeFeatured?: boolean;
  primaryTypes?: PrimaryType[];
  excludedSecondaryTypes?: SecondaryType[];
}

/** PATCH body: an absent field is unchanged, null resets it to the global setting. */
export interface FilterPatch {
  primaryTypes?: PrimaryType[] | null;
  excludedSecondaryTypes?: SecondaryType[] | null;
  includeFeatured?: boolean | null;
}

export type SyncState = 'pending' | 'synced';

export function syncStateOf(s: Pick<Subscription, 'lastSyncedAt'>): SyncState {
  return s.lastSyncedAt === null ? 'pending' : 'synced';
}
