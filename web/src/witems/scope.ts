/** Class mount() puts on the app root and the portal (Shadow DOM has no :root/html/body to match). */
export const ROOT_SELECTOR = ".punisher-root";

/** Rewrites :root, html and body in a CSS selector to the app root (vite.witems.config.ts). */
export function scopeSelector(selector: string): string {
  return selector
    .replace(/:root\b/g, ROOT_SELECTOR)
    .replace(/(^|[\s>+~,(])(?:html|body)(?=$|[\s>+~,.:[)])/g, `$1${ROOT_SELECTOR}`);
}
