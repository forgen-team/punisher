import { describe, expect, it } from "vitest";
import type { CheckDetail, CheckItem, CheckResultItem, CheckSession, CheckTurn } from "@/src/shared/api/punisher";
import {
  canSend,
  currentItemId,
  disputeTarget,
  isShownItem,
  itemTurns,
  judgedTurn,
  earliestNextReview,
  objectParticle,
  orderedItems,
  progressAt,
  progressOf,
  progressRatio,
  resultRows,
  retryableTurnId,
  selfGradeStep,
  showComposer,
  showItemReview,
  showReveal,
  stepMeta,
  stopCounts,
} from "../lib/checkView";

const NOW = Date.parse("2026-11-13T12:00:00+09:00");

function session(patch: Partial<CheckSession> = {}): CheckSession {
  return {
    id: "s1",
    kind: "check",
    grader: "llm",
    conceptId: "os.page-cache",
    lectureId: "l1",
    itemIds: ["i1", "i2", "i3"],
    cursor: 0,
    status: "active",
    result: null,
    createdAt: "",
    updatedAt: "",
    finishedAt: null,
    ...patch,
  };
}

function item(id: string, patch: Partial<CheckItem> = {}): CheckItem {
  return { id, objectiveId: `os.page-cache#${id}`, format: "recall", prompt: `q ${id}`, done: false, ...patch };
}

let seq = 0;
function turn(patch: Partial<CheckTurn>): CheckTurn {
  seq += 1;
  return { id: `t${seq}`, itemId: "i1", seq, role: "tutor", move: "ask", text: "", status: "ready", aided: false, reusedItem: false, createdAt: "", ...patch };
}

function detail(patch: Partial<CheckDetail> = {}): CheckDetail {
  return {
    session: session(),
    items: [item("i1"), item("i2"), item("i3", { objectiveId: "", format: "integrative" })],
    turns: [],
    current: { itemId: "i1", answers: 0, canAnswer: true, canReveal: true, canSelfGrade: false },
    pending: false,
    mastery: [],
    ...patch,
  };
}

describe("현재 문항·진행", () => {
  it("문항은 itemIds 순서로 정렬한다", () => {
    const d = detail({ items: [item("i3"), item("i1"), item("i2")] });
    expect(orderedItems(d).map((it) => it.id)).toEqual(["i1", "i2", "i3"]);
  });

  it("current 가 있으면 그 문항, 없으면 cursor 위치", () => {
    expect(currentItemId(detail())).toBe("i1");
    expect(currentItemId(detail({ current: null, session: session({ cursor: 1 }) }))).toBe("i2");
  });

  it("active 가 아니면 현재 문항이 없다", () => {
    expect(currentItemId(detail({ session: session({ status: "finished", cursor: 3 }), current: null }))).toBeNull();
  });

  it("진행은 cursor+1/N, 끝난 세션은 N/N", () => {
    expect(progressOf(detail({ session: session({ cursor: 1 }) }))).toEqual({ index: 2, total: 3 });
    expect(progressOf(detail({ session: session({ status: "finished", cursor: 3 }) }))).toEqual({ index: 3, total: 3 });
  });
});

describe("턴", () => {
  it("첫 물음(ask)은 문항 머리에 있으므로 빼고 seq 순으로", () => {
    const a = turn({ move: "ask", text: "q i1" });
    const b = turn({ role: "learner", move: "answer", text: "답" });
    const c = turn({ move: "narrow", verdict: "partial" });
    expect(itemTurns([c, a, b], "i1").map((t) => t.id)).toEqual([b.id, c.id]);
  });

  it("판정 턴은 outcome 이 붙은 마지막 턴(실패·이의 제외)", () => {
    const judged = turn({ move: "next", verdict: "incorrect", outcome: "failed" });
    const disputeTurn = turn({ move: "dispute", verdict: "correct", outcome: "unaided_correct" });
    expect(judgedTurn([judged, disputeTurn], "i1")?.id).toBe(judged.id);
  });
});

describe("버튼 노출", () => {
  it("입력창은 active + current 일 때만", () => {
    expect(showComposer(detail())).toBe(true);
    expect(showComposer(detail({ current: null }))).toBe(false);
    expect(showComposer(detail({ session: session({ status: "finished" }) }))).toBe(false);
  });

  it("'그냥 알려줘'는 canReveal 이고 실행 중이 아닐 때만", () => {
    expect(showReveal(detail())).toBe(true);
    expect(showReveal(detail({ pending: true }))).toBe(false);
    expect(showReveal(detail({ current: { itemId: "i1", answers: 2, canAnswer: true, canReveal: false, canSelfGrade: false } }))).toBe(false);
  });

  it("보내기는 답할 차례 + 내용 있음 + 실행 중 아님", () => {
    expect(canSend(detail(), "  답 ")).toBe(true);
    expect(canSend(detail(), "   ")).toBe(false);
    expect(canSend(detail({ pending: true }), "답")).toBe(false);
    expect(canSend(detail({ current: { itemId: "i1", answers: 1, canAnswer: false, canReveal: false, canSelfGrade: false } }), "답")).toBe(false);
  });

  it("'다시 시도'는 마지막 턴이 실패한 튜터 턴일 때만", () => {
    const answer = turn({ role: "learner", move: "answer" });
    const failed = turn({ move: "", status: "failed" });
    expect(retryableTurnId(detail({ turns: [answer, failed] }))).toBe(failed.id);
    expect(retryableTurnId(detail({ turns: [failed, turn({ role: "learner", move: "answer" })] }))).toBeNull();
    expect(retryableTurnId(detail({ turns: [failed], pending: true }))).toBeNull();
  });
});

describe("'내 답이 맞아요'", () => {
  const done = item("i1", { done: true, outcome: "failed" });
  const judged = () => turn({ move: "next", verdict: "incorrect", outcome: "failed" });

  it("끝난 목표 문항의 이의 대상 판정에 붙는다", () => {
    const target = judged();
    expect(disputeTarget(detail({ turns: [target] }), done, NOW)?.id).toBe(target.id);
  });

  it("통합 문항에는 두지 않는다", () => {
    const integrative = item("i3", { objectiveId: "", done: true, outcome: "failed" });
    expect(disputeTarget(detail({ turns: [turn({ itemId: "i3", verdict: "incorrect", outcome: "failed" })] }), integrative, NOW)).toBeNull();
  });

  it("공개(revealed)·혼자 맞힘·이미 이의한 문항·실행 중에는 두지 않는다", () => {
    expect(disputeTarget(detail({ turns: [turn({ move: "reveal", verdict: "revealed", outcome: "failed" })] }), done, NOW)).toBeNull();
    const correct = item("i1", { done: true, outcome: "unaided_correct" });
    expect(disputeTarget(detail({ turns: [turn({ move: "next", verdict: "correct", outcome: "unaided_correct" })] }), correct, NOW)).toBeNull();
    expect(disputeTarget(detail({ turns: [judged(), turn({ move: "dispute", status: "ready" })] }), done, NOW)).toBeNull();
    expect(disputeTarget(detail({ turns: [judged()], pending: true }), done, NOW)).toBeNull();
  });

  it("실패한 이의 턴만 있으면 다시 낼 수 있다", () => {
    const target = judged();
    expect(disputeTarget(detail({ turns: [target, turn({ move: "dispute", status: "failed" })] }), done, NOW)?.id).toBe(target.id);
  });

  it("끝난 세션은 24시간 안에만", () => {
    const recent = session({ status: "finished", finishedAt: new Date(NOW - 23 * 3600_000).toISOString() });
    const old = session({ status: "finished", finishedAt: new Date(NOW - 25 * 3600_000).toISOString() });
    expect(disputeTarget(detail({ session: recent, current: null, turns: [judged()] }), done, NOW)).not.toBeNull();
    expect(disputeTarget(detail({ session: old, current: null, turns: [judged()] }), done, NOW)).toBeNull();
  });
});

describe("결과 요약", () => {
  it("통합 문항 줄은 빼고 목표 원문·대기 여부를 붙인다", () => {
    const entry = { levelBefore: 0, levelAfter: 3, missing: null, anchor: "s1", misconception: "", nextReviewAt: null, lapses: 0 };
    const d = detail({
      session: session({
        status: "finished",
        result: {
          items: [
            { ...entry, itemId: "i1", objectiveId: "os.page-cache#i1", outcome: "unaided_correct" },
            { ...entry, itemId: "i3", objectiveId: "", outcome: "unaided_correct" },
          ],
        },
      }),
      mastery: [
        {
          objectiveId: "os.page-cache#i1", conceptId: "os.page-cache", text: "페이지 캐시를 설명한다", depth: 1, ordinal: 1, level: 3, box: 1,
          gradStep: 0, best: "unaided", integrativeOk: false, lapses: 0, nextReviewAt: null, stale: false, queued: true, lastGrader: "llm", updatedAt: null,
        },
      ],
    });
    const rows = resultRows(d);
    expect(rows).toHaveLength(1);
    expect(rows[0]).toMatchObject({ text: "페이지 캐시를 설명한다", queued: true });
  });
});

describe("복습(자기 채점)", () => {
  const selfSession = (patch: Partial<CheckSession> = {}) => session({ kind: "review", grader: "self", lectureId: "", conceptId: "", ...patch });

  it("진행 줄은 '3/12 · 복습' — 문항이 없으면 제목만", () => {
    expect(stepMeta(progressAt(detail({ session: selfSession({ cursor: 1 }) }), "i2"), "복습")).toBe("2/3 · 복습");
    expect(stepMeta(progressOf(detail()), "확인하기")).toBe("1/3 · 확인하기");
    expect(stepMeta({ index: 0, total: 0 }, "확인하기")).toBe("확인하기");
  });

  it("끝낸 문항을 보는 중에는 그 문항 자리, 막대는 그 문항까지 찬다", () => {
    const d = detail({ session: selfSession({ cursor: 2 }) });
    expect(progressAt(d, "i2")).toEqual({ index: 2, total: 3 });
    expect(progressAt(d, "없음")).toEqual({ index: 3, total: 3 });
    expect(progressRatio(2, 4, false)).toBe(0.25);
    expect(progressRatio(2, 4, true)).toBe(0.5);
    expect(progressRatio(1, 0, true)).toBe(0);
  });

  it("자기 채점 단계: 답 쓰기 → 채점 고르기, 실행 중·튜터 채점·현재 문항 없음은 null", () => {
    const answerStep = { itemId: "i1", answers: 0, canAnswer: true, canReveal: false, canSelfGrade: false };
    const gradeStep = { itemId: "i1", answers: 1, canAnswer: false, canReveal: false, canSelfGrade: true };
    expect(selfGradeStep(detail({ session: selfSession(), current: answerStep }))).toBe("answer");
    expect(selfGradeStep(detail({ session: selfSession(), current: gradeStep }))).toBe("grade");
    expect(selfGradeStep(detail({ session: selfSession(), current: gradeStep, pending: true }))).toBeNull();
    expect(selfGradeStep(detail({ session: selfSession(), current: null }))).toBeNull();
    expect(selfGradeStep(detail({ session: session({ kind: "review", grader: "llm" }), current: gradeStep }))).toBeNull();
    expect(selfGradeStep(detail({ session: selfSession(), current: { ...gradeStep, canSelfGrade: false } }))).toBeNull();
  });

  it("답을 보낸 자기 채점 문항은 끝나기 전에도 모범답·기준을 연다", () => {
    expect(showItemReview(item("i1"))).toBe(false);
    expect(showItemReview(item("i1", { answer: "메모리에 남는다", must: ["메모리"] }))).toBe(true);
    expect(showItemReview(item("i1", { done: true }))).toBe(true);
  });

  it("자기 채점 판정에는 '내 답이 맞아요'를 두지 않는다", () => {
    const d = detail({
      session: selfSession(),
      items: [item("i1", { done: true, outcome: "unaided_partial" })],
      turns: [turn({ itemId: "i1", role: "learner", move: "self_grade", verdict: "partial", outcome: "unaided_partial", grader: "self" })],
    });
    expect(disputeTarget(d, d.items[0], NOW)).toBeNull();
  });
});

describe("isShownItem", () => {
  it("shows done, current, or answered items only", () => {
    expect(isShownItem(item("a", { done: true }), "b", 0)).toBe(true);
    expect(isShownItem(item("b"), "b", 0)).toBe(true);
    expect(isShownItem(item("c"), "b", 2)).toBe(true);
    expect(isShownItem(item("d"), "b", 0)).toBe(false);
  });
});

describe("끝 화면·그만두기", () => {
  const result = (patch: Partial<CheckResultItem>): CheckResultItem => ({
    itemId: "i1", objectiveId: "o1", outcome: "unaided_correct", levelBefore: 1, levelAfter: 2, missing: [], anchor: "", misconception: "", nextReviewAt: null, lapses: 0, ...patch,
  });

  it("다음 복습일은 대기 아닌 줄 중 가장 이른 날짜", () => {
    expect(earliestNextReview([result({ nextReviewAt: "2026-12-05T00:00:00Z" }), result({ nextReviewAt: "2026-12-02T00:00:00Z" }), result({ nextReviewAt: "2026-11-30T00:00:00Z", queued: true })])).toBe("2026-12-02T00:00:00Z");
    expect(earliestNextReview([result({})])).toBeNull();
    expect(earliestNextReview(undefined)).toBeNull();
  });

  it("그만두기 확인 창의 개수 — 끝낸 문항 + 지금 문항에 답한 것", () => {
    const d = detail({ items: [item("i1", { done: true }), item("i2"), item("i3")] });
    expect(stopCounts(d, false)).toEqual({ answered: 1, remaining: 2 });
    expect(stopCounts(d, true)).toEqual({ answered: 2, remaining: 1 });
  });

  it("목적격 조사는 받침으로 가른다", () => {
    expect(objectParticle("복습")).toBe("을");
    expect(objectParticle("확인하기")).toBe("를");
    expect(objectParticle("종합 문제")).toBe("를");
    expect(objectParticle("짧은 확인")).toBe("을");
  });
});
