/**
 * Theme preference (ported from @witems/config/theme)
 * Jotai-based theme management with localStorage persistence
 */

import { atom } from 'jotai';
import { atomWithStorage } from 'jotai/utils';

export type ThemePreference = 'light' | 'dark' | 'system';
export type ResolvedTheme = 'light' | 'dark';

/**
 * User's theme preference stored in localStorage
 */
export const themePreferenceAtom = atomWithStorage<ThemePreference>(
  'punisher-theme',
  'system'
);

/**
 * System's preferred color scheme
 */
export const systemThemeAtom = atom<ResolvedTheme>(() => {
  if (typeof window === 'undefined') return 'light';
  return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
});

/**
 * Resolved theme based on preference and system settings
 */
export const resolvedThemeAtom = atom<ResolvedTheme>((get) => {
  const preference = get(themePreferenceAtom);
  if (preference !== 'system') return preference;
  return get(systemThemeAtom);
});

/**
 * Action atom to toggle between light and dark themes
 */
export const toggleThemeAtom = atom(
  null,
  (get, set) => {
    const current = get(resolvedThemeAtom);
    set(themePreferenceAtom, current === 'light' ? 'dark' : 'light');
  }
);

/**
 * Action atom to cycle through all theme options
 */
export const cycleThemeAtom = atom(
  null,
  (get, set) => {
    const current = get(themePreferenceAtom);
    const next: ThemePreference =
      current === 'light' ? 'dark' :
      current === 'dark' ? 'system' : 'light';
    set(themePreferenceAtom, next);
  }
);

/**
 * Theme icon based on current preference
 */
export const themeIconAtom = atom((get) => {
  const preference = get(themePreferenceAtom);
  switch (preference) {
    case 'light': return 'sun';
    case 'dark': return 'moon';
    case 'system': return 'monitor';
  }
});

/**
 * Theme label for UI display
 */
export const themeLabelAtom = atom((get) => {
  const preference = get(themePreferenceAtom);
  switch (preference) {
    case 'light': return 'Light';
    case 'dark': return 'Dark';
    case 'system': return 'System';
  }
});
