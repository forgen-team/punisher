import type { RecheckCard, ReviewSessionRef, TodayResponse } from "@/src/shared/api/punisher";

/** [복습 시작]이 여는 세션 — 복습이 있으면 복습, 없으면 첫 확인 대기 강의의 확인, 그것도 없으면 종합 문제(D12 ①). */
export type ReviewTarget = { kind: "review" } | { kind: "check"; lectureId: string } | { kind: "synthesis" } | { kind: "none" };

export interface ReviewCounts {
  /** 만기 복습 + 섞을 '이미 앎' 확인 */
  review: number;
  /** 확인하기 안 한 강의 + 강의가 있는 '다시 확인'(만기 아님) */
  confirm: number;
  cards: number;
  synthesis: number;
  minutes: number;
}

export type ReviewBundleState =
  /** 할 것 없음 — "지금 볼 복습은 없어요"(행동 없음) */
  | { kind: "none"; counts: ReviewCounts }
  /** 오늘 세션을 중간에 나감 — "이어서 복습 · n개 남음" */
  | { kind: "partial"; counts: ReviewCounts; left: number }
  /** 오늘 세션을 마침 — "오늘 복습을 다 했어요 · d/t" + [오늘 더 하기] */
  | { kind: "done"; counts: ReviewCounts; done: number; total: number }
  /** 쉬는 날 — "오늘은 쉬는 날이에요" + [그래도 복습] */
  | { kind: "rest"; counts: ReviewCounts; target: ReviewTarget }
  /** 숫자 줄 "복습 n · 확인 대기 n · 카드 n · 종합 1" + [복습 시작] */
  | { kind: "due"; counts: ReviewCounts; target: ReviewTarget };

export interface ReviewInputs {
  today: Pick<TodayResponse, "review" | "unchecked" | "recheck" | "synthesis" | "session"> | undefined;
  /** 오늘 복습 세션(GET /reviews/summary 의 today) */
  session: ReviewSessionRef | null | undefined;
  /** 오늘 볼 카드 수(GET /study/activity 의 today.cardsDue) */
  cardsDue: number;
  restToday: boolean;
}

/** 강의가 있는 '다시 확인'(만기 아님)만 확인 대기로 센다 — 만기면 복습 수에 이미 들어 있고, 강의가 없으면 추천으로 간다. */
function confirmRecheck(recheck: readonly RecheckCard[] | undefined): RecheckCard[] {
  return (recheck ?? []).filter((card) => !card.due && Boolean(card.lectureId));
}

/** 강의가 없는 '다시 확인' — 오늘의 추천에 강의 행으로 더한다(§5-1). */
export function recheckWithoutLecture(recheck: readonly RecheckCard[] | undefined): RecheckCard[] {
  return (recheck ?? []).filter((card) => !card.due && !card.lectureId);
}

export function reviewCounts({ today, cardsDue }: Pick<ReviewInputs, "today" | "cardsDue">): ReviewCounts {
  const review = today?.review;
  const session = today?.session;
  const minutes = session ? session.reviewMinutes + session.checkMinutes : (review?.estimatedMinutes ?? 0);
  return {
    review: (review?.due ?? 0) + (review?.verify ?? 0),
    confirm: (today?.unchecked?.length ?? 0) + confirmRecheck(today?.recheck).length,
    cards: Math.max(0, cardsDue),
    synthesis: today?.synthesis?.due ? 1 : 0,
    minutes: Number.isFinite(minutes) && minutes > 0 ? Math.max(1, Math.round(minutes)) : 0,
  };
}

export function reviewTarget(counts: ReviewCounts, today: ReviewInputs["today"]): ReviewTarget {
  if (counts.review > 0) { return { kind: "review" }; }
  const lectureId = today?.unchecked?.[0]?.lectureId ?? confirmRecheck(today?.recheck)[0]?.lectureId;
  if (lectureId) { return { kind: "check", lectureId }; }
  if (counts.synthesis > 0) { return { kind: "synthesis" }; }
  return { kind: "none" };
}

/** 처리할 것 상자의 합계 — 복습 + 확인 대기 + 카드. */
export function reviewTotal(counts: ReviewCounts): number {
  return counts.review + counts.confirm + counts.cards;
}

/**
 * 복습 묶음 상태 — 순서: 중간에 나감 → 다 함 → 할 것 없음 → 쉬는 날 → 숫자 줄.
 * 카드만 남은 날은 숫자 줄("카드 n")에 [카드로 복습]만 보인다(target none).
 */
export function reviewBundleState(inputs: ReviewInputs): ReviewBundleState {
  const counts = reviewCounts(inputs);
  const { session } = inputs;
  if (session?.status === "active" && session.total > session.done) {
    return { kind: "partial", counts, left: session.total - session.done };
  }
  if (session?.status === "finished") {
    return { kind: "done", counts, done: session.done, total: session.total };
  }
  if (counts.review + counts.confirm + counts.synthesis + counts.cards === 0) { return { kind: "none", counts }; }
  const target = reviewTarget(counts, inputs.today);
  if (inputs.restToday) { return { kind: "rest", counts, target }; }
  return { kind: "due", counts, target };
}
