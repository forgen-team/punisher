import { describe, expect, it } from "vitest";
import { legacyHashPath } from "../legacyHash";

describe("legacyHashPath (옛 해시 주소 → 경로, 한 릴리스용)", () => {
  it("turns the old section-menu hash on basePath into the section path", () => {
    expect(legacyHashPath("/", "", "#/map")).toBe("/map");
    expect(legacyHashPath("/", "", "#/history")).toBe("/history");
    expect(legacyHashPath("/", "", "#/goals")).toBe("/goals");
    expect(legacyHashPath("/", "", "#/")).toBe("/");
    expect(legacyHashPath("/", "", "#/?today")).toBe("/?today");
    expect(legacyHashPath("/", "", "#/lectures")).toBe("/");
    expect(legacyHashPath("/", "", "#/unknown")).toBe("/");
  });

  it("keeps the path and query of the old path#section address and drops the hash", () => {
    expect(legacyHashPath("/map", "?concept=a", "#/map")).toBe("/map?concept=a");
    expect(legacyHashPath("/lectures/l1", "?at=p3", "#/lectures")).toBe("/lectures/l1?at=p3");
  });

  it("leaves addresses without an old hash alone", () => {
    expect(legacyHashPath("/map", "", "")).toBeNull();
    expect(legacyHashPath("/", "", "#note")).toBeNull();
  });
});
