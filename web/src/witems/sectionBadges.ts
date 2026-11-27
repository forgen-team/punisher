import { useEffect } from "react";
import { witemsRuntime } from "@/src/shared/config/runtime";
import { useDueCount } from "@/src/widgets/punisher/nav";

/**
 * 셸 섹션 메뉴(manifest nav.items — internal/witems/app.go) 숫자: 복습 + 확인 대기 + 카드를 '오늘' 섹션에 단다
 * (단독판 탭 '오늘' 옆과 같은 값). 키는 그 섹션의 href 경로("/", contract §3 setBadges). setBadges 가 없는 셸이면 아무것도 하지 않는다.
 */
export function useSectionBadges() {
  const due = useDueCount();
  useEffect(() => {
    witemsRuntime()?.setBadges?.({ "/": due });
  }, [due]);
}
