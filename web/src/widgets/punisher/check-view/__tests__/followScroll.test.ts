import { describe, expect, it } from "vitest";
import type { CheckTurn } from "@/src/shared/api/punisher";
import { turnBody } from "../lib/checkView";
import { FOLLOW_WINDOW_MS, checkItemAnchorId, isScrollKey, shouldFollow } from "../lib/followScroll";

describe("shouldFollow", () => {
  const now = 1_000_000;
  it("follows only right after the learner acted and did not scroll since", () => {
    expect(shouldFollow({ actedAt: now - 2000, scrolledAt: null, now })).toBe(true);
    expect(shouldFollow({ actedAt: now - 2000, scrolledAt: now - 5000, now })).toBe(true);
    expect(shouldFollow({ actedAt: now - 2000, scrolledAt: now - 1000, now })).toBe(false);
    expect(shouldFollow({ actedAt: null, scrolledAt: null, now })).toBe(false);
    expect(shouldFollow({ actedAt: now - FOLLOW_WINDOW_MS - 1, scrolledAt: null, now })).toBe(false);
    expect(shouldFollow({ actedAt: now - FOLLOW_WINDOW_MS, scrolledAt: null, now })).toBe(true);
  });
});

describe("isScrollKey", () => {
  it("counts page keys outside text fields only", () => {
    const div = { tagName: "DIV", isContentEditable: false } as unknown as EventTarget;
    const textarea = { tagName: "TEXTAREA", isContentEditable: false } as unknown as EventTarget;
    expect(isScrollKey("PageUp", div)).toBe(true);
    expect(isScrollKey("ArrowUp", null)).toBe(true);
    expect(isScrollKey("ArrowUp", textarea)).toBe(false);
    expect(isScrollKey("a", div)).toBe(false);
  });

  it("anchors the item head by id", () => {
    expect(checkItemAnchorId("i1")).toBe("check-item-i1");
  });
});

describe("turnBody", () => {
  const base: CheckTurn = { id: "t1", itemId: "i1", seq: 2, role: "tutor", move: "reveal", text: "모범답 본문", status: "ready", verdict: "revealed", aided: false, reusedItem: false, createdAt: "" };
  it("hides the model answer on the reveal turn (shown once in 모범답) and keeps other tutor text", () => {
    expect(turnBody(base)).toBe("");
    expect(turnBody({ ...base, move: "narrow", verdict: "partial", text: "어디에 남을까요?" })).toBe("어디에 남을까요?");
    expect(turnBody({ ...base, move: "wrap", text: "정리" })).toBe("정리");
  });
});
