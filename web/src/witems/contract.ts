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
  /** Optional: older shells don't have it — the app then keeps its own settings link (LearnNav). */
  setActions?(actions: AppAction[] | null): void;
  portal: HTMLElement;
}
