import { describe, expect, it } from "vitest";
import { isShellOwnedDeclaration, scopeSelector } from "../scope";

describe("scopeSelector", () => {
  it.each([
    [":root", ".punisher-root"],
    [":root:not([data-theme])", ".punisher-root:not([data-theme])"],
    ["html", ".punisher-root"],
    ["body", ".punisher-root"],
    ["html[data-theme=\"dark\"] .x", ".punisher-root[data-theme=\"dark\"] .x"],
    ["body > main", ".punisher-root > main"],
    [":is(html) a", ":is(.punisher-root) a"],
  ])("%s → %s", (input, output) => {
    expect(scopeSelector(input)).toBe(output);
  });

  it.each(["[data-theme=\"dark\"]", ".html-preview", ".tbody", "td.body-cell", "#htmlish", ":host"])("leaves %s alone", (sel) => {
    expect(scopeSelector(sel)).toBe(sel);
  });
});

describe("isShellOwnedDeclaration", () => {
  it("drops the shell's layout variables from the Witems build", () => {
    expect(isShellOwnedDeclaration("--app-header-offset")).toBe(true);
    expect(isShellOwnedDeclaration("--app-bottom-offset")).toBe(true);
    expect(isShellOwnedDeclaration("--app-topbar-h")).toBe(true);
    expect(isShellOwnedDeclaration("--safe-top")).toBe(false);
    expect(isShellOwnedDeclaration("--color-bg")).toBe(false);
  });
});
