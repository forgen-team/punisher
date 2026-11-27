import { useActivity } from "@/src/entities/punisher";

/**
 * 앱 안 배지(W5a): 지금 볼 복습·카드 수(쉬는 날은 0). 단독판은 '강의' 탭에, Witems 는 셸 섹션 메뉴 '강의'
 * 항목(ctx.setBadges {"/": n} — witems/sectionBadges.ts)에 보인다.
 */
export function useDueCount(): number {
  const activity = useActivity().data;
  return activity && !activity.restToday ? activity.today.reviewsDue + activity.today.cardsDue : 0;
}
