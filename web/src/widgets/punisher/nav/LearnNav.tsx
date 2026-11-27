import { useEffect, type MouseEvent } from "react";
import { LinkTabs } from "@forgen-team/ui";
import { useRouter } from "@/src/shared/lib/router";
import { useT } from "@/src/shared/i18n";
import { useLocation } from "react-router-dom";
import { HREFS, SEGMENTS, rememberTab, segmentOf, tabHref, tabScroll, type LearnSegment } from "./lib/tabMemory";
import { useDueCount } from "./lib/useDueCount";
import { messages } from "./i18n";

/**
 * 단독판 섹션 탭 줄 [오늘·지도·기록](policy/20 §2 "제목 아래 탭 줄" — 단독판은 앱 이름 줄 아래). 탭 줄에는 탭만 둔다.
 * 섹션 밖 화면(설정 등)에서도 보이고 현재 표시만 없다. 탭은 링크(⌘/Ctrl·가운데 클릭 새 탭), 화살표는 포커스만.
 * 다른 섹션에 갔다 돌아오면 그 섹션의 마지막 주소·스크롤로, 지금 섹션을 다시 누르면 첫 화면으로.
 * '오늘' 옆 숫자 = 복습 + 확인 대기 + 카드(useDueCount). Witems 셸 안에서는 쓰지 않는다 — 섹션 메뉴는 셸이
 * manifest nav.items 로 그리고, 같은 숫자를 witems/sectionBadges.ts 가 보낸다.
 */
export function LearnNav() {
  const { pathname, search } = useLocation();
  const router = useRouter();
  const segment = segmentOf(pathname);
  const t = useT(messages);
  const due = useDueCount();
  useEffect(() => {
    if (segment) { rememberTab(segment, `${pathname}${search}`); }
  }, [segment, pathname, search]);

  function select(next: LearnSegment) {
    if (segment) { rememberTab(segment, `${pathname}${search}`, window.scrollY); }
    if (next === segment) {
      router.push(HREFS[next]);
      return;
    }
    router.push(tabHref(next), { restoreScroll: tabScroll(next) });
  }

  const hrefOf = (value: LearnSegment) => (value === segment ? HREFS[value] : tabHref(value));
  // ui LinkTabs 는 진짜 링크(⌘/Ctrl·가운데 클릭 새 탭)다. 보통 클릭만 감싼 곳에서 먼저 받아 탭 기억(주소·스크롤)으로 옮긴다.
  function handleClickCapture(event: MouseEvent<HTMLDivElement>) {
    if (event.button !== 0 || event.metaKey || event.ctrlKey || event.shiftKey || event.altKey) { return; }
    const link = event.target instanceof Element ? event.target.closest("a") : null;
    const next = SEGMENTS.find((value) => link?.getAttribute("href") === hrefOf(value));
    if (!next) { return; }
    event.preventDefault();
    event.stopPropagation();
    select(next);
  }

  return (
    <div className="min-w-0" onClickCapture={handleClickCapture}>
      <LinkTabs
        label={t.tabsLabel}
        items={SEGMENTS.map((value) => ({
          value,
          label: t[value],
          href: hrefOf(value),
          count: value === "today" && due > 0 ? due : undefined,
        }))}
        value={segment ?? ""}
      />
    </div>
  );
}
