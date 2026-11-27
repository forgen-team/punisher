import { beforeEach, describe, expect, it } from "vitest";
import { isSectionRoot, rememberTab, resetTabMemory, segmentOf, tabHref, tabScroll } from "../lib/tabMemory";

describe("tab memory", () => {
  beforeEach(() => resetTabMemory());

  it("returns the tab's first screen when nothing is remembered", () => {
    expect(tabHref("map")).toBe("/map");
    expect(tabScroll("map")).toBeUndefined();
  });

  it("restores the last address and scroll of a tab", () => {
    rememberTab("history", "/history?area=net");
    rememberTab("history", "/history?area=net", 420);
    expect(tabHref("history")).toBe("/history?area=net");
    expect(tabScroll("history")).toBe(420);
  });

  it("drops the old scroll when the address changes", () => {
    rememberTab("map", "/map?concept=a", 300);
    rememberTab("map", "/map?concept=b");
    expect(tabHref("map")).toBe("/map?concept=b");
    expect(tabScroll("map")).toBeUndefined();
  });

  it("maps paths to sections like manifest nav.items match", () => {
    expect(segmentOf("/")).toBe("today");
    for (const path of ["/lectures/1", "/check/9", "/cards", "/inquiries/3", "/instant/j1", "/start"]) {
      expect(segmentOf(path)).toBe("today");
    }
    expect(segmentOf("/map")).toBe("map");
    expect(segmentOf("/chapters/new")).toBe("map");
    expect(segmentOf("/history")).toBe("history");
    expect(segmentOf("/settings")).toBeNull();
    expect(segmentOf("/settings/prompts")).toBeNull();
    // 앞부분만 같은 다른 경로는 섹션이 아니다
    expect(segmentOf("/mapping")).toBeNull();
    expect(segmentOf("/cardsx")).toBeNull();
  });

  it("knows the section first screens (no ←)", () => {
    expect(isSectionRoot("/")).toBe(true);
    expect(isSectionRoot("/map")).toBe(true);
    expect(isSectionRoot("/history")).toBe(true);
    expect(isSectionRoot("/lectures/1")).toBe(false);
    expect(isSectionRoot("/settings")).toBe(false);
  });
});
