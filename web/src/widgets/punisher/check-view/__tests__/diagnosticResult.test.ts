import { describe, expect, it } from "vitest";
import type { CheckDetail, CheckResultItem, CheckTurn } from "@/src/shared/api/punisher";
import { firstAnswerOf, isRecorded, resultRows, suggestRelearn } from "../lib/checkView";

function row(patch: Partial<CheckResultItem> = {}): CheckResultItem {
  return {
    itemId: "i1", objectiveId: "c#1", outcome: "unaided_correct", levelBefore: 0, levelAfter: 3,
    missing: [], anchor: "", misconception: "", nextReviewAt: null, lapses: 0, ...patch,
  };
}

function learner(itemId: string, seq: number, text: string, move: CheckTurn["move"] = "answer"): CheckTurn {
  return { id: `t${seq}`, itemId, seq, role: "learner", move, text, status: "ready", aided: false, reusedItem: false, createdAt: "" };
}

function detail(patch: Partial<CheckDetail> = {}): CheckDetail {
  return {
    session: {
      id: "s1", kind: "check", grader: "llm", conceptId: "c", lectureId: "l1", itemIds: ["i1", "i2"], cursor: 2,
      status: "finished", result: { items: [row(), row({ itemId: "i2", objectiveId: "c#2", outcome: "failed" })] },
      createdAt: "", updatedAt: "",
    },
    items: [],
    turns: [learner("i1", 3, "나중 답"), learner("i1", 1, "메모리에서 읽어요"), learner("i2", 2, "디스크")],
    current: null,
    pending: false,
    mastery: [],
    ...patch,
  };
}

describe("처음 예측 → 지금 답", () => {
  it("같은 목표의 예측과 그 문항의 첫 답을 나란히 둔다", () => {
    const rows = resultRows(detail({
      predictions: [{ objectiveId: "c#1", prompt: "어디서 읽을까요?", answer: "디스크", dontKnow: false, at: "" }],
    }));
    expect(rows[0]).toMatchObject({ prediction: { answer: "디스크" }, answer: "메모리에서 읽어요" });
    expect(rows[1]).toMatchObject({ prediction: null, answer: "디스크" });
  });

  it("첫 답은 seq 순 첫 학습자 답(이의 등은 제외)", () => {
    expect(firstAnswerOf([learner("i1", 2, "둘째"), learner("i1", 1, "", "dispute")], "i1")).toBe("둘째");
    expect(firstAnswerOf([], "i1")).toBe("");
  });
});

describe("사전 진단 세션", () => {
  it("예측·배치는 기록만 하므로 결과 줄이 없다", () => {
    const placement = detail();
    placement.session = { ...placement.session, kind: "placement", grader: "none" };
    expect(isRecorded(placement)).toBe(true);
    expect(resultRows(placement)).toEqual([]);
    expect(isRecorded(detail())).toBe(false);
  });

  it("'다시 배우기'는 반복 실패 또는 이미 앎 확인 실패에서 제안한다", () => {
    expect(suggestRelearn(row({ lapses: 2 }))).toBe(true);
    expect(suggestRelearn(row({ relearn: true, outcome: "failed" }))).toBe(true);
    expect(suggestRelearn(row({ lapses: 1 }))).toBe(false);
  });
});
