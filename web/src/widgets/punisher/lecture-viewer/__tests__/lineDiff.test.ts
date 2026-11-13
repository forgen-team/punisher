import { describe, expect, it } from "vitest";
import { diffLines } from "../lib/lineDiff";

describe("diffLines", () => {
  it("marks a changed line on both sides", () => {
    expect(diffLines("a\nb\nc", "a\nB\nc")).toEqual({ removed: [2], added: [2] });
  });
  it("marks inserted and deleted lines only", () => {
    expect(diffLines("a\nc", "a\nb\nc")).toEqual({ removed: [], added: [2] });
    expect(diffLines("a\nb\nc", "a\nc")).toEqual({ removed: [2], added: [] });
  });
  it("handles a full rewrite and trailing additions", () => {
    expect(diffLines("x", "y\nz")).toEqual({ removed: [1], added: [1, 2] });
    expect(diffLines("a", "a\nb\nc")).toEqual({ removed: [], added: [2, 3] });
  });
});
