/** Witems app contract 1 — the shell's AppContext (witems docs/APP-CONTRACT.md §3). */
export type Unmount = () => void;
export type AppTheme = "light" | "dark";

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
  portal: HTMLElement;
}
