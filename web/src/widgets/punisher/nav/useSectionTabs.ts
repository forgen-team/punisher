import { useEffect, type MouseEvent } from "react";
import { useLocation } from "react-router-dom";
import type { AppSection } from "@forgen-team/ui";
import { useRouter } from "@/src/shared/lib/router";
import { useT } from "@/src/shared/i18n";
import { HREFS, SEGMENTS, rememberTab, segmentOf, tabHref, tabScroll, type LearnSegment } from "./lib/tabMemory";
import { useDueCount } from "./lib/useDueCount";
import { messages } from "./i18n";

export interface SectionTabs {
  /** AppFrame sections — 오늘·지도·기록, href 는 탭 기억(마지막 주소), '오늘' count = 할 일 수 */
  sections: AppSection[];
  /** 지금 섹션(섹션 밖 화면이면 undefined — 현재 표시 없음) */
  current: LearnSegment | undefined;
  /** 탭 줄 이름(AppFrame sectionsLabel) */
  label: string;
  /** AppFrame 을 감싼 곳에 단다 — 탭의 보통 클릭을 받아 탭 기억(주소·스크롤)으로 옮긴다 */
  onClickCapture: (event: MouseEvent<HTMLElement>) => void;
}

function isPlainClick(event: MouseEvent): boolean {
  return event.button === 0 && !event.metaKey && !event.ctrlKey && !event.shiftKey && !event.altKey;
}

/**
 * 단독판 섹션 탭 [오늘·지도·기록] 정보(policy/20 §2 — 탭 줄은 AppFrame 이 그린다).
 * 다른 섹션에 갔다 돌아오면 그 섹션의 마지막 주소·스크롤로, 지금 섹션을 다시 누르면 첫 화면으로(policy/20 §4).
 * 탭은 진짜 링크라 ⌘/Ctrl·가운데 클릭 새 탭은 그대로 두고, 보통 클릭만 받아 스크롤까지 되살린다.
 * Witems 셸 안에서는 섹션 메뉴를 셸이 manifest nav.items 로 그리고, 같은 숫자를 witems/sectionBadges.ts 가 보낸다.
 */
export function useSectionTabs(): SectionTabs {
  const { pathname, search } = useLocation();
  const router = useRouter();
  const segment = segmentOf(pathname);
  const t = useT(messages);
  const due = useDueCount();
  useEffect(() => {
    if (segment) { rememberTab(segment, `${pathname}${search}`); }
  }, [segment, pathname, search]);

  const hrefOf = (value: LearnSegment) => (value === segment ? HREFS[value] : tabHref(value));

  function select(next: LearnSegment) {
    if (segment) { rememberTab(segment, `${pathname}${search}`, window.scrollY); }
    if (next === segment) {
      router.push(HREFS[next]);
      return;
    }
    router.push(tabHref(next), { restoreScroll: tabScroll(next) });
  }

  function onClickCapture(event: MouseEvent<HTMLElement>) {
    if (!isPlainClick(event)) { return; }
    const link = event.target instanceof Element ? event.target.closest("a") : null;
    // 탭 줄(이름이 t.tabsLabel 인 nav) 안의 링크만 — 상단 바 브랜드("/")·본문 링크는 그대로 둔다.
    if (!link || link.closest("nav")?.getAttribute("aria-label") !== t.tabsLabel) { return; }
    const next = SEGMENTS.find((value) => link.getAttribute("href") === hrefOf(value));
    if (!next) { return; }
    event.preventDefault();
    event.stopPropagation();
    select(next);
  }

  return {
    sections: SEGMENTS.map((value) => ({
      value,
      label: t[value],
      href: hrefOf(value),
      count: value === "today" && due > 0 ? due : undefined,
    })),
    current: segment ?? undefined,
    label: t.tabsLabel,
    onClickCapture,
  };
}
