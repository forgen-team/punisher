import postcss from "postcss";
import { describe, expect, it } from "vitest";
import brandCss from "@/src/styles/brand.css?raw";
import tokensCss from "@/src/styles/design-tokens.css?raw";
import { declaredCustomProperties, scopeToRoot } from "../cssScope";

const SHELL_TOKENS = declaredCustomProperties(tokensCss, brandCss);
const ORANGE = ["#f97316", "#ea580c", "#c2410c", "#fb923c", "#fff1e6", "#2a1708"];

function witemsCss(css: string): string {
  return postcss([scopeToRoot(SHELL_TOKENS)]).process(css, { from: undefined }).css;
}

describe("Witems 빌드 CSS", () => {
  it("collects the shell's token names from design-tokens.css and brand.css", () => {
    for (const name of ["--color-bg", "--color-accent", "--color-border-strong", "--radius-md", "--shadow-xs", "--font-sans", "--text-sm", "--z-modal", "--brand-orange"]) {
      expect(SHELL_TOKENS.has(name), name).toBe(true);
    }
    expect(SHELL_TOKENS.has("--app-topbar-h")).toBe(false);
  });

  it("drops every token declaration and the orange accent, leaving no empty rules", () => {
    const out = witemsCss(`${tokensCss}\n${brandCss}`);
    expect(out).not.toMatch(/--[\w-]+\s*:/);
    for (const hex of ORANGE) { expect(out.toLowerCase()).not.toContain(hex); }
    expect(out).not.toMatch(/\{\s*\}/);
  });

  it("keeps rules that only use tokens, scoped to the app root", () => {
    const out = witemsCss(":root { --color-bg: #fff; color-scheme: light; } body { background: var(--color-bg); } .x { --app-header-offset: 1px; color: red; }");
    expect(out).toContain(".punisher-root { color-scheme: light; }");
    expect(out).toContain(".punisher-root { background: var(--color-bg); }");
    expect(out).toContain(".x { color: red; }");
    expect(out).not.toContain("--app-header-offset");
  });

  it("ignores names that only appear in comments", () => {
    expect(declaredCustomProperties("/* --fake: 1 */ :root { --real: 2; }")).toEqual(new Set(["--real"]));
  });
});
