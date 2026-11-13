import { describe, expect, it } from "vitest";
import type { CheckGrade, CheckTurn } from "@/src/shared/api/punisher";
import { itemMisconceptions, misconceptionButtonState } from "../lib/checkView";

function grade(misconception: string): CheckGrade {
  return { met: [], missing: [], quote: "", misconception, confidence: 0.9 };
}

function turn(id: string, seq: number, patch: Partial<CheckTurn>): CheckTurn {
  return { id, itemId: "i1", seq, role: "tutor", move: "narrow", text: "", status: "ready", aided: false, reusedItem: false, createdAt: "", ...patch };
}

describe("itemMisconceptions", () => {
  it("lists each misconception of the item's judgements once, in order, with its turn", () => {
    const turns = [
      turn("t3", 3, { move: "next", verdict: "incorrect", grade: grade("항상 디스크") }),
      turn("t1", 1, { verdict: "partial", grade: grade(" 항상 디스크 ") }),
      turn("t2", 2, { role: "learner", move: "answer", grade: grade("학습자 턴") }),
      turn("t4", 4, { itemId: "i2", verdict: "incorrect", grade: grade("다른 문항") }),
      turn("t5", 5, { move: "dispute", verdict: "correct", grade: grade("이의") }),
      turn("t6", 6, { status: "failed", grade: grade("실패") }),
      turn("t7", 7, { move: "explain", verdict: "incorrect", grade: grade("캐시는 무한하다") }),
      turn("t8", 8, { verdict: "correct", grade: grade("") }),
    ];
    expect(itemMisconceptions(turns, "i1")).toEqual([
      { turnId: "t1", text: "항상 디스크" },
      { turnId: "t7", text: "캐시는 무한하다" },
    ]);
    expect(itemMisconceptions([], "i1")).toEqual([]);
  });
});

describe("misconceptionButtonState", () => {
  it("is acknowledged once the turn is acknowledged, pending only for the turn being sent", () => {
    const acknowledged = new Set(["t1"]);
    expect(misconceptionButtonState("t1", acknowledged, null)).toBe("acknowledged");
    expect(misconceptionButtonState("t1", acknowledged, "t1")).toBe("acknowledged");
    expect(misconceptionButtonState("t2", acknowledged, "t2")).toBe("pending");
    expect(misconceptionButtonState("t2", acknowledged, "t3")).toBe("idle");
    expect(misconceptionButtonState("t2", new Set(), null)).toBe("idle");
  });
});
