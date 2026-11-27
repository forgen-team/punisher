/** Witems app contract 1 — the shell's AppContext (witems docs/APP-CONTRACT.md §3). */
export type Unmount = () => void;
export type AppTheme = "light" | "dark";

/**
 * Page action the app hands to the shell (contract 1, optional — witems docs/UI-PAGE-LAYOUT.md §1).
 * The shell draws them in the top bar (phone: one icon + ⋯, desktop: up to two text buttons + ⋯);
 * kind "settings" is the app's settings entry (last ⋯ item / last button). href is basePath-relative.
 */
export interface AppAction {
  id: string;
  label: string;
  /** lucide icon name (kebab-case) */
  icon?: string;
  href?: string;
  onSelect?: () => void;
  kind?: "default" | "settings";
  danger?: boolean;
}

/**
 * Command for the shell's ⌘K search (contract 1, optional — APP-CONTRACT §3 "명령"). Up to 50, replaced on
 * every call, null clears; the shell drops invalid ones and clears them when the app leaves. href is basePath-relative.
 */
export interface AppCommand {
  id: string;
  label: string;
  group?: string;
  keywords?: string[];
  href?: string;
  onSelect?: () => void;
  /** display only — the shell binds no key */
  shortcut?: string;
}

export interface AppContext {
  contract: 1;
  app: string;
  basePath: string;
  apiBase: string;
  readonly path: string;
  readonly search: string;
  onRouteChange(callback: (path: string, search: string) => void): () => void;
  navigate(to: string, options?: { replace?: boolean }): void;
  user: { id: string; email: string; name: string | null; role: string };
  readonly theme: AppTheme;
  onThemeChange(callback: (theme: AppTheme) => void): () => void;
  setTitle(title: string | null): void;
  /** Optional: one line under the title (date, state) — null clears. Older shells lack it (witems#54). */
  setSubtitle?(subtitle: string | null): void;
  /**
   * Optional: numbers on the section menu (manifest nav.items) — keys are section paths { "/": n }, unknown
   * items and n ≤ 0 are ignored.
   */
  setBadges?(badges: Record<string, number>): void;
  /** Optional: older shells don't have it — the app then can't show its page actions inside the shell. */
  setActions?(actions: AppAction[] | null): void;
  /** Optional: ⌘K commands. The shell owns ⌘K (captures it); older shells lack this and the app keeps its own ⌘K. */
  setCommands?(commands: AppCommand[] | null): void;
  portal: HTMLElement;
}
