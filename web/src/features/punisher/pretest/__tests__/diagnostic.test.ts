import { describe, expect, it } from "vitest";
import type { CheckDetail, CheckItem } from "@/src/shared/api/punisher";
import { choicesOf, currentDiagnosticItem, diagnosticProgress, promptMarkdown } from "../lib/diagnostic";

function item(id: string, patch: Partial<CheckItem> = {}): CheckItem {
  return { id, objectiveId: `c#${id}`, format: "choice", prompt: `q ${id}`, done: false, material: { choices: ["가", " ", "나", "다"] }, ...patch };
}

function detail(patch: Partial<CheckDetail> = {}): CheckDetail {
  return {
    session: { id: "p1", kind: "pretest", grader: "none", conceptId: "c", lectureId: "l1", itemIds: ["a", "b"], cursor: 1, status: "active", result: null, createdAt: "", updatedAt: "" },
    items: [item("a", { done: true }), item("b", { format: "predict", material: {} })],
    turns: [],
    current: { itemId: "b", answers: 0, canAnswer: true, canReveal: false, canSelfGrade: false },
    pending: false,
    mastery: [],
    ...patch,
  };
}

describe("예측 질문", () => {
  it("답할 차례인 문항만 낸다(준비 중·실행 중·끝남·답한 뒤는 없음)", () => {
    expect(currentDiagnosticItem(detail())?.id).toBe("b");
    expect(currentDiagnosticItem(undefined)).toBeNull();
    expect(currentDiagnosticItem(detail({ pending: true }))).toBeNull();
    expect(currentDiagnosticItem(detail({ current: null }))).toBeNull();
    expect(currentDiagnosticItem(detail({ current: { itemId: "b", answers: 1, canAnswer: false, canReveal: false, canSelfGrade: false } }))).toBeNull();
    const finished = detail();
    finished.session = { ...finished.session, status: "finished" };
    expect(currentDiagnosticItem(finished)).toBeNull();
  });

  it("보기는 choice 문항에서만, 빈 보기는 뺀다", () => {
    expect(choicesOf(item("a"))).toEqual(["가", "나", "다"]);
    expect(choicesOf(item("b", { format: "predict" }))).toEqual([]);
  });

  it("진행 n/N 과 자료 코드 블록", () => {
    expect(diagnosticProgress(detail())).toEqual({ index: 2, total: 2 });
    expect(promptMarkdown(item("c", { material: { code: "x := 1", lang: "go" } }))).toBe("q c\n\n```go\nx := 1\n```");
    expect(promptMarkdown(item("d"))).toBe("q d");
  });
});
