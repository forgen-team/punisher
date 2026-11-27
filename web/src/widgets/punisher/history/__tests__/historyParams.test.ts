import { describe, expect, it } from "vitest";
import { historyHref, resolveHistoryView } from "../lib/historyParams";

describe("resolveHistoryView", () => {
  it("defaults to the learn tab and ignores unknown values", () => {
    expect(resolveHistoryView(null, null)).toEqual({ tab: "learn", kind: "", legacy: false });
    expect(resolveHistoryView("nope", "memo")).toEqual({ tab: "learn", kind: "", legacy: false });
  });

  it("keeps the three new tabs; the kind only counts on the notes tab", () => {
    expect(resolveHistoryView("review", "memo")).toEqual({ tab: "review", kind: "", legacy: false });
    expect(resolveHistoryView("notes", "lq")).toEqual({ tab: "notes", kind: "lq", legacy: false });
    expect(resolveHistoryView("notes", "bogus")).toEqual({ tab: "notes", kind: "", legacy: false });
  });

  it("maps the old six tabs onto the new ones", () => {
    expect(resolveHistoryView("lectures", null)).toEqual({ tab: "learn", kind: "", legacy: true });
    expect(resolveHistoryView("learned", null)).toEqual({ tab: "learn", kind: "", legacy: true });
    expect(resolveHistoryView("checks", null)).toEqual({ tab: "review", kind: "", legacy: true });
    expect(resolveHistoryView("inquiries", null)).toEqual({ tab: "notes", kind: "inquiry", legacy: true });
    expect(resolveHistoryView("questions", null)).toEqual({ tab: "notes", kind: "lq", legacy: true });
  });
});

describe("historyHref", () => {
  it("carries the search term across tabs but area/kind only on their own tab", () => {
    expect(historyHref("/history", { tab: "review", q: "캐시", area: "network", kind: "memo" })).toBe(`/history?tab=review&q=${encodeURIComponent("캐시")}`);
    expect(historyHref("/history", { tab: "learn", area: "network" })).toBe("/history?tab=learn&area=network");
    expect(historyHref("/history", { tab: "notes", kind: "lq" })).toBe("/history?tab=notes&kind=lq");
  });
});
