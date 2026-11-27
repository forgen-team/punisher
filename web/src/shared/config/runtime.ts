/**
 * Where this UI runs: the standalone app (same origin as the Go server, /v1) or
 * inside the Witems shell as a Module Federation app (src/witems/mount.tsx sets it).
 * In Witems the API is /apps-api/punisher/v1, overlays render in the shell's
 * Shadow DOM portal, and the screen title goes to the shell's top bar.
 */
export interface WitemsRuntime {
  /** ctx.apiBase, e.g. "/apps-api/punisher/" */
  apiBase: string;
  /** element inside the app's Shadow DOM for popovers, sheets and toasts */
  portal: HTMLElement;
  /** shell top-bar title (null = the app's title) */
  setTitle(title: string | null): void;
  /** shell page actions (ctx.setActions) — undefined when the shell predates it */
  setActions?: (actions: ShellAction[] | null) => void;
  /** shell ⌘K commands (ctx.setCommands) — undefined when the shell predates it */
  setCommands?: (commands: ShellCommand[] | null) => void;
  /** shell sidebar sub-menu numbers (ctx.setBadges, keys = manifest nav.items href) — undefined when the shell predates it */
  setBadges?: (badges: Record<string, number>) => void;
}

/** Same shape as the contract's AppCommand (witems/contract.ts). */
export interface ShellCommand {
  id: string;
  label: string;
  group?: string;
  keywords?: string[];
  href?: string;
  onSelect?: () => void;
  shortcut?: string;
}

/** Same shape as the contract's AppAction (witems/contract.ts) — kept here so shared code doesn't import the mount. */
export interface ShellAction {
  id: string;
  label: string;
  icon?: string;
  href?: string;
  onSelect?: () => void;
  kind?: "default" | "settings";
  danger?: boolean;
}

let runtime: WitemsRuntime | null = null;

export function setWitemsRuntime(next: WitemsRuntime | null): void {
  runtime = next;
}

export function isWitems(): boolean {
  return runtime !== null;
}

export function witemsRuntime(): WitemsRuntime | null {
  return runtime;
}

/** Does the shell draw page actions (ctx.setActions)? Standalone and older shells: false. */
export function shellHasActions(): boolean {
  return typeof runtime?.setActions === "function";
}

/**
 * Does the shell draw the app's sub-menu (manifest nav.items) in its sidebar? Shells that do also take
 * ctx.setBadges, so that is the sign. Then the app's own tabs (LearnNav) would repeat it. Standalone: false.
 */
export function shellHasSubNav(): boolean {
  return typeof runtime?.setBadges === "function";
}

/** Does the shell take ⌘K commands (ctx.setCommands)? Then the shell owns ⌘K and the app must not bind it. */
export function shellHasCommands(): boolean {
  return typeof runtime?.setCommands === "function";
}

/**
 * 단독판 화면의 가운데 좁은 칸(mx-auto max-w-2xl). Witems 셸은 본문 자리 폭을 제한하지 않으므로
 * (witems AppHost APP_CONTAINER_CLASS — 오너 결정 2026-11-19) Witems 에서는 빈 문자열 = 왼쪽 정렬 전체 폭.
 * 읽기 글 폭(LectureLayout 의 --reading-measure)은 이것과 별개로 두 곳 모두 유지한다.
 */
export function standaloneColumnClass(): string {
  return runtime ? "" : "mx-auto max-w-2xl";
}

/** API base without a trailing slash: "/v1" standalone, "/apps-api/punisher/v1" in Witems. */
export function apiBase(): string {
  return runtime ? `${runtime.apiBase.replace(/\/+$/, "")}/v1` : "/v1";
}

/** Target for createPortal: document.body standalone, the Shadow DOM portal in Witems. */
export function portalRoot(): HTMLElement {
  return runtime?.portal ?? document.body;
}
