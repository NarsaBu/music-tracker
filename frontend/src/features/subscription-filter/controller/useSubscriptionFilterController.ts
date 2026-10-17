import type { PrimaryType, SecondaryType } from '@/entities/release/model';
import { toggleItem } from '@/entities/settings/model';
import type { FilterPatch } from '@/entities/subscription/model';
import { useSubscriptionQuery, useUpdateFilterMutation } from '@/entities/subscription/queries';

/**
 * Per-artist filter override (US-5, US-6). Every change is saved at once; a field
 * switched back to "global" is sent as null and follows the settings again.
 */
export function useSubscriptionFilterController(mbid: string) {
  const subscription = useSubscriptionQuery(mbid);
  const update = useUpdateFilterMutation(mbid);
  const filter = subscription.data?.filter;
  const patch = (p: FilterPatch) => update.mutate(p);

  const primaryTypes = filter?.primaryTypes.value ?? [];
  const excluded = (filter?.excludedSecondaryTypes.value ?? []) as SecondaryType[];

  return {
    ready: !!filter,
    saving: update.isPending,
    error: update.isError ? 'Не удалось сохранить фильтр' : undefined,

    primaryTypes,
    ownTypes: filter?.primaryTypes.overridden ?? false,
    excludedSecondaryTypes: excluded,
    ownSecondary: filter?.excludedSecondaryTypes.overridden ?? false,
    includeFeatured: filter?.includeFeatured.value ?? false,
    ownFeatured: filter?.includeFeatured.overridden ?? false,

    // Taking a field over starts from its current effective value.
    onOwnTypesChange: (own: boolean) => patch({ primaryTypes: own ? primaryTypes : null }),
    onTogglePrimaryType: (t: PrimaryType) => patch({ primaryTypes: toggleItem(primaryTypes, t) }),
    onOwnSecondaryChange: (own: boolean) =>
      patch({ excludedSecondaryTypes: own ? excluded : null }),
    // Switches show "include"; the model stores exclusions.
    onToggleSecondaryType: (t: SecondaryType) =>
      patch({ excludedSecondaryTypes: toggleItem(excluded, t) }),
    onOwnFeaturedChange: (own: boolean) =>
      patch({ includeFeatured: own ? (filter?.includeFeatured.value ?? false) : null }),
    onIncludeFeaturedChange: (v: boolean) => patch({ includeFeatured: v }),
  };
}

export type SubscriptionFilterViewModel = ReturnType<typeof useSubscriptionFilterController>;
