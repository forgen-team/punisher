import type { ReviewSessionRef, TodayReview } from "@/src/shared/api/punisher";

export type ReviewCardState =
  | { kind: "hidden" }
  /** 만기 0 + 오늘 세션을 마침 → 접힌 완료 카드 */
  | { kind: "done"; done: number; total: number }
  | {
      kind: "due";
      due: number;
      minutes: number;
      lectureMinutes: number;
      /** 밀림(backlogged)일 때 "밀린 복습 n개"의 n, 아니면 null */
      backlog: number | null;
      /** 오늘 세션을 마쳤으면 '오늘 더 하기', 아니면 '복습 시작' */
      action: "start" | "more";
    };

/**
 * 오늘 화면 복습 카드 — 숫자는 today.review, 완료 여부는 오늘 세션(GET /reviews/summary 의 today).
 * 만기(+ 섞을 '이미 앎' 확인)가 있으면 카드, 없고 오늘 세션을 마쳤으면 접힌 완료, 둘 다 없으면 숨긴다.
 */
export function reviewCardState(review: TodayReview | undefined, today: ReviewSessionRef | null | undefined): ReviewCardState {
  if (!review) { return { kind: "hidden" }; }
  const finished = today?.status === "finished";
  const verify = review.verify ?? 0;
  if (review.due + verify > 0) {
    const backlog = review.overdue > 0 ? review.overdue : review.due;
    return {
      kind: "due",
      due: review.due + verify,
      minutes: review.estimatedMinutes,
      lectureMinutes: review.lectureMinutes,
      backlog: review.backlogged ? backlog : null,
      action: finished ? "more" : "start",
    };
  }
  if (finished && today) { return { kind: "done", done: today.done, total: today.total }; }
  return { kind: "hidden" };
}
