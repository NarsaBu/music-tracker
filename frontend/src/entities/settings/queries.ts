import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { httpClient } from '@/shared/api';
import { createSettingsApi } from './api';
import type { Settings, SettingsState } from './model';

const settingsApi = createSettingsApi(httpClient);

export const settingsKeys = {
  all: ['settings'] as const,
};

export function useSettingsQuery() {
  return useQuery({ queryKey: settingsKeys.all, queryFn: settingsApi.get });
}

export function useSaveSettingsMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (s: Settings) => settingsApi.save(s),
    onSuccess: (state) => queryClient.setQueryData(settingsKeys.all, state),
  });
}

export function useSendTestEmailMutation() {
  return useMutation({ mutationFn: settingsApi.sendTestEmail });
}

export function useRotateFeedTokenMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: settingsApi.rotateFeedToken,
    onSuccess: (feedUrl) =>
      queryClient.setQueryData<SettingsState>(
        settingsKeys.all,
        (old) =>
          old && { ...old, settings: { ...old.settings, rss: { ...old.settings.rss, feedUrl } } },
      ),
  });
}
