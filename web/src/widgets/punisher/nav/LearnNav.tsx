import { useEffect, type MouseEvent } from "react";
import { LinkTabs } from "@forgen-team/ui";
import { useRouter } from "@/src/shared/lib/router";
import { useT } from "@/src/shared/i18n";
import { useLocation } from "react-router-dom";
import { HREFS, SEGMENTS, rememberTab, segmentOf, tabHref, tabScroll, type LearnSegment } from "./lib/tabMemory";
import { useDueCount } from "./lib/useDueCount";
import { messages } from "./i18n";

/**
 * 단독판 학습 화면 상단: 제목(데스크톱만) + 탭 [강의·지도·기록] — 탭 줄에는 탭만 둔다(UI-PAGE-LAYOUT §2).
 * 탭은 링크(⌘/Ctrl·가운데 클릭 새 탭), 화살표는 포커스만. 다른 탭에 갔다 돌아오면 그 탭의 마지막 주소·스크롤로,
 * 지금 탭을 다시 누르면 처음 화면으로. 강의 뷰어·즉석 강의·설정 화면에서는 렌더하지 않는다.
 * Witems 셸 안에서는 쓰지 않는다 — 섹션 메뉴는 셸이 manifest nav.items 로 그리고(D8), 복습 수는
 * witems/sectionBadges.ts 가 셸 '강의' 항목 숫자로 보낸다.
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
  if (!segment) { return null; }

  function select(next: LearnSegment) {
    if (!segment) { return; }
    rememberTab(segment, `${pathname}${search}`, window.scrollY);
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
    <div className="shrink-0">
      {/* 단독판 셸 상단 바는 화면 제목을 보이지 않으므로 md 이상에서 여기 둔다. */}
      <h1 className="hidden truncate py-4 pb-2 text-xl font-semibold tracking-tight text-fg md:block md:text-2xl">{t.title}</h1>
      <div className="flex min-w-0 items-center gap-2" onClickCapture={handleClickCapture}>
        <LinkTabs
          label={t.tabsLabel}
          items={SEGMENTS.map((value) => ({
            value,
            label: t[value],
            href: hrefOf(value),
            count: value === "lectures" && due > 0 ? due : undefined,
          }))}
          value={segment}
        />
      </div>
    </div>
  );
}
