import { beforeEach, describe, expect, it } from "vitest";
import { rememberTab, resetTabMemory, segmentOf, tabHref, tabScroll } from "../lib/tabMemory";

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

  it("maps paths to tabs", () => {
    expect(segmentOf("/")).toBe("lectures");
    expect(segmentOf("/goals")).toBe("map");
    expect(segmentOf("/lectures/1")).toBeNull();
  });
});
