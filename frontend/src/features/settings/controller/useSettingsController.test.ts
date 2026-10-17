import { act, renderHook, waitFor } from '@testing-library/react';
import { http, HttpResponse } from 'msw';
import { DEFAULT_SETTINGS, type SettingsState } from '@/entities/settings/model';
import { settingsWire } from '@/test/fixtures';
import { createWrapper } from '@/test/queryWrapper';
import { server } from '@/test/server';
import { useSettingsController } from './useSettingsController';

const initial: SettingsState = {
  settings: DEFAULT_SETTINGS,
  lastDigestAt: null,
  nextDigestAt: null,
};

function renderController(state: SettingsState = initial) {
  const { Wrapper } = createWrapper();
  return renderHook(() => useSettingsController(state), { wrapper: Wrapper });
}

describe('useSettingsController', () => {
  it('starts clean with the loaded settings', () => {
    const { result } = renderController();
    expect(result.current.settings).toEqual(DEFAULT_SETTINGS);
    expect(result.current.isDirty).toBe(false);
    expect(result.current.canSave).toBe(false);
    expect(result.current.nextDigestText).toBe('Дайджест выключен');
  });

  it('toggles primary types and becomes dirty', () => {
    const { result } = renderController();

    act(() => result.current.onTogglePrimaryType('single'));
    expect(result.current.settings.filters.primaryTypes).not.toContain('single');
    expect(result.current.isDirty).toBe(true);

    act(() => result.current.onTogglePrimaryType('single'));
    expect(result.current.isDirty).toBe(false);
  });

  it('switching a secondary type on removes it from exclusions', () => {
    const { result } = renderController();
    act(() => result.current.onToggleSecondaryType('Compilation'));
    expect(result.current.settings.filters.excludedSecondaryTypes).not.toContain('Compilation');
  });

  it('blocks saving when email is enabled without a valid address', () => {
    const { result } = renderController();

    act(() => result.current.onEmailEnabledChange(true));
    expect(result.current.errors.emailTo).toBeDefined();
    expect(result.current.canSave).toBe(false);

    act(() => result.current.onEmailToChange('me@example.com'));
    expect(result.current.canSave).toBe(true);
  });

  it('saves to the server and takes the returned schedule', async () => {
    let body: { email: { enabled: boolean; to: string } } | undefined;
    server.use(
      http.put('/api/v1/settings', async ({ request }) => {
        body = (await request.json()) as typeof body;
        return HttpResponse.json(
          settingsWire({ emailEnabled: true, to: 'me@example.com', next: '2026-10-18T09:00:00Z' }),
        );
      }),
    );
    const { result } = renderController();
    act(() => result.current.onEmailEnabledChange(true));
    act(() => result.current.onEmailToChange('me@example.com'));
    act(() => result.current.onSave());

    await waitFor(() => expect(result.current.isDirty).toBe(false));
    expect(body?.email).toMatchObject({ enabled: true, to: 'me@example.com' });
    expect(result.current.nextDigestText).toMatch(/^Следующий дайджест: 18\.10\.2026/);
    expect(result.current.canTestEmail).toBe(true);
  });

  it('shows field errors returned by the server', async () => {
    server.use(
      http.put('/api/v1/settings', () =>
        HttpResponse.json(
          {
            title: 'Validation failed',
            errors: [{ field: 'email.digest.timezone', message: 'unknown time zone' }],
          },
          { status: 422 },
        ),
      ),
    );
    const { result } = renderController();
    act(() => result.current.onTimezoneChange('Mars/Olympus'));
    act(() => result.current.onSave());
    await waitFor(() => expect(result.current.saveError).toMatch(/email\.digest\.timezone/));
    expect(result.current.isDirty).toBe(true);
  });

  it('test email goes to the saved address only', async () => {
    server.use(
      http.post('/api/v1/settings/email/test', () => new HttpResponse(null, { status: 204 })),
    );
    const saved: SettingsState = {
      ...initial,
      settings: {
        ...DEFAULT_SETTINGS,
        email: { ...DEFAULT_SETTINGS.email, enabled: true, to: 'me@example.com' },
      },
    };
    const { result } = renderController(saved);
    expect(result.current.canTestEmail).toBe(true);

    act(() => result.current.onEmailToChange('other@example.com'));
    expect(result.current.canTestEmail).toBe(false);
    act(() => result.current.onReset());

    act(() => result.current.onSendTestEmail());
    await waitFor(() =>
      expect(result.current.testEmailText).toBe('Письмо отправлено на me@example.com'),
    );
  });

  it('reports an SMTP outage on the test email', async () => {
    server.use(
      http.post('/api/v1/settings/email/test', () =>
        HttpResponse.json({ title: 'Upstream unavailable' }, { status: 503 }),
      ),
    );
    const saved: SettingsState = {
      ...initial,
      settings: { ...DEFAULT_SETTINGS, email: { ...DEFAULT_SETTINGS.email, to: 'me@example.com' } },
    };
    const { result } = renderController(saved);
    act(() => result.current.onSendTestEmail());
    await waitFor(() => expect(result.current.testEmailFailed).toBe(true));
    expect(result.current.testEmailText).toMatch(/SMTP/);
  });

  it('rotating the feed link keeps the form clean', async () => {
    server.use(
      http.post('/api/v1/settings/feed-token/rotate', () =>
        HttpResponse.json({ feed_url: 'http://localhost:5173/feeds/new.atom' }),
      ),
    );
    const { result } = renderController();
    act(() => result.current.onRotateFeedToken());
    await waitFor(() =>
      expect(result.current.settings.rss.feedUrl).toBe('http://localhost:5173/feeds/new.atom'),
    );
    expect(result.current.isDirty).toBe(false);
  });
});
