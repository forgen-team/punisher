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

/** API base without a trailing slash: "/v1" standalone, "/apps-api/punisher/v1" in Witems. */
export function apiBase(): string {
  return runtime ? `${runtime.apiBase.replace(/\/+$/, "")}/v1` : "/v1";
}

/** Target for createPortal: document.body standalone, the Shadow DOM portal in Witems. */
export function portalRoot(): HTMLElement {
  return runtime?.portal ?? document.body;
}
