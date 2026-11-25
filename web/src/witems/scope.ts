/** Class mount() puts on the app root and the portal (Shadow DOM has no :root/html/body to match). */
export const ROOT_SELECTOR = ".punisher-root";

/** Rewrites :root, html and body in a CSS selector to the app root (vite.witems.config.ts). */
export function scopeSelector(selector: string): string {
  return selector
    .replace(/:root\b/g, ROOT_SELECTOR)
    .replace(/(^|[\s>+~,(])(?:html|body)(?=$|[\s>+~,.:[)])/g, `$1${ROOT_SELECTOR}`);
}

/**
 * 셸이 정하는 레이아웃 변수 — Witems 상단·하단 바 높이는 셸 globals.css 가 <html> 에 두고 Shadow DOM 으로
 * 이어진다. 앱 CSS 가 .punisher-root 에 다시 정의하면 셸 값을 가려(단독판의 48px 상단 바·하단 탭 없음)
 * 강의 상단 바가 셸 바에 겹치거나 아래가 하단 탭에 가린다. Witems 빌드에서는 이 선언을 뺀다.
 */
export const SHELL_OWNED_VARS: ReadonlySet<string> = new Set(["--app-topbar-h", "--app-header-offset", "--app-bottom-offset"]);

export function isShellOwnedDeclaration(prop: string): boolean {
  return SHELL_OWNED_VARS.has(prop);
}
