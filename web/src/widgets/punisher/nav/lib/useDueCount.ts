import { useActivity, useToday } from "@/src/entities/punisher";

/** 섹션 배지 수 = 복습 + 확인 대기 + 카드(쉬는 날은 0). 순수 함수 — 테스트·두 셸이 같은 셈을 쓴다. */
export function dueTotal(input: { restToday: boolean; reviewsDue: number; cardsDue: number; unchecked: number }): number {
  if (input.restToday) { return 0; }
  return Math.max(0, input.reviewsDue) + Math.max(0, input.cardsDue) + Math.max(0, input.unchecked);
}

/**
 * '오늘' 섹션 배지(재설계 §1·§6): 복습 + 확인 대기(GET /today 의 unchecked) + 카드. 단독판은 탭 '오늘' 옆,
 * Witems 는 셸 섹션 메뉴 '오늘' 항목(ctx.setBadges {"/": n} — witems/sectionBadges.ts)에 같은 값.
 */
export function useDueCount(): number {
  const activity = useActivity().data;
  const unchecked = useToday().data?.unchecked?.length ?? 0;
  if (!activity) { return 0; }
  return dueTotal({
    restToday: activity.restToday,
    reviewsDue: activity.today.reviewsDue,
    cardsDue: activity.today.cardsDue,
    unchecked,
  });
}
