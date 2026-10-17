import { act, renderHook } from '@testing-library/react';
import { DEFAULT_SETTINGS } from '@/entities/settings/model';
import { useSettingsController } from './useSettingsController';

describe('useSettingsController', () => {
  it('starts clean with defaults', () => {
    const { result } = renderHook(() => useSettingsController());
    expect(result.current.settings).toEqual(DEFAULT_SETTINGS);
    expect(result.current.isDirty).toBe(false);
    expect(result.current.canSave).toBe(false);
  });

  it('toggles primary types and becomes dirty', () => {
    const { result } = renderHook(() => useSettingsController());

    act(() => result.current.onTogglePrimaryType('single'));
    expect(result.current.settings.filters.primaryTypes).not.toContain('single');
    expect(result.current.isDirty).toBe(true);

    act(() => result.current.onTogglePrimaryType('single'));
    expect(result.current.settings.filters.primaryTypes).toContain('single');
    expect(result.current.isDirty).toBe(false);
  });

  it('switching a secondary type on removes it from exclusions', () => {
    const { result } = renderHook(() => useSettingsController());

    act(() => result.current.onToggleSecondaryType('Compilation'));
    expect(result.current.settings.filters.excludedSecondaryTypes).not.toContain('Compilation');
  });

  it('blocks saving when email is enabled without a valid address', () => {
    const { result } = renderHook(() => useSettingsController());

    act(() => result.current.onEmailEnabledChange(true));
    expect(result.current.errors.emailTo).toBeDefined();
    expect(result.current.canSave).toBe(false);

    act(() => result.current.onEmailToChange('me@example.com'));
    expect(result.current.errors.emailTo).toBeUndefined();
    expect(result.current.canSave).toBe(true);
  });

  it('save makes the draft the new baseline, reset reverts to it', () => {
    const { result } = renderHook(() => useSettingsController());

    act(() => result.current.onFrequencyChange('weekly'));
    act(() => result.current.onSave());
    expect(result.current.isDirty).toBe(false);

    act(() => result.current.onFrequencyChange('daily'));
    act(() => result.current.onReset());
    expect(result.current.settings.email.digest.frequency).toBe('weekly');
  });
});
