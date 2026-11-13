import type { RecheckCard, Recommendation } from "@/src/shared/api/punisher";

export type TodayRow = { type: "recheck"; card: RecheckCard } | { type: "recommendation"; item: Recommendation };

/** 추천 목록 — "○○ 다시 확인" 카드는 첫 목표 경로 추천 앞에(목표 경로 추천이 없으면 맨 앞). */
export function todayRows(recommendations: Recommendation[], recheck: RecheckCard[] | undefined): TodayRow[] {
  const cards: TodayRow[] = (recheck ?? []).map((card) => ({ type: "recheck", card }));
  const rows: TodayRow[] = recommendations.map((item) => ({ type: "recommendation", item }));
  if (cards.length === 0) { return rows; }
  const at = recommendations.findIndex((item) => item.kind === "goal_path");
  const index = at < 0 ? 0 : at;
  return [...rows.slice(0, index), ...cards, ...rows.slice(index)];
}

export type RecheckAction = "review" | "check" | "map";

/** 카드를 누르면 — 만기된 목표가 있으면 오늘 복습, 아니면 그 개념 강의로 확인하기, 강의가 없으면 지도. */
export function recheckAction(card: RecheckCard): RecheckAction {
  if (card.due) { return "review"; }
  return card.lectureId ? "check" : "map";
}
