import { describe, expect, it } from "vitest";
import { initialAppPath, segmentFromShell, segmentOfHash, toShellPath } from "../sections";

describe("witems sections (manifest nav.items)", () => {
  it("puts the section hash on tab screens only", () => {
    expect(toShellPath("/")).toBe("/#/");
    expect(toShellPath("/?at=3")).toBe("/?at=3#/");
    expect(toShellPath("/map?concept=a")).toBe("/map?concept=a#/map");
    expect(toShellPath("/goals")).toBe("/goals#/goals");
    expect(toShellPath("/history?area=net")).toBe("/history?area=net#/history");
    expect(toShellPath("/lectures/l1")).toBe("/lectures/l1#/lectures");
    expect(toShellPath("/settings")).toBe("/settings");
    expect(toShellPath("/search?q=tcp")).toBe("/search?q=tcp");
  });

  it("maps sub-menu hashes to tabs", () => {
    expect(segmentOfHash("#/")).toBe("lectures");
    expect(segmentOfHash("#/map")).toBe("map");
    expect(segmentOfHash("#/goals")).toBe("map");
    expect(segmentOfHash("#/history")).toBe("history");
    expect(segmentOfHash("#/lectures")).toBeNull();
    expect(segmentOfHash("")).toBeNull();
  });

  it("follows the path when the hash agrees, the sub-menu when it does not", () => {
    // 뒤로가기·새로고침: 경로와 해시가 맞다
    expect(segmentFromShell("/map", "#/map")).toBeNull();
    expect(segmentFromShell("/lectures/l1", "#/lectures")).toBeNull();
    expect(segmentFromShell("/settings", "")).toBeNull();
    // 다른 하위 화면에서 하위 메뉴: 셸은 /apps/punisher#/history 로 옮긴다
    expect(segmentFromShell("/", "#/history")).toBe("history");
    expect(segmentFromShell("/", "#/")).toBeNull();
  });

  it("opens the sub-menu tab on mount", () => {
    expect(initialAppPath("/", "", "#/map")).toBe("/map");
    expect(initialAppPath("/history", "?area=net", "#/history")).toBe("/history?area=net");
    expect(initialAppPath("/lectures/l1", "", "")).toBe("/lectures/l1");
  });
});
