import { useEffect } from "react";
import { Link, useRouter } from "@/src/shared/lib/router";
import { PageHeader, SegmentedTabs } from "@/src/shared/ui/kit";
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

  return (
    <div className="shrink-0">
      {/* 단독판 셸 상단 바는 화면 제목을 보이지 않으므로 md 이상에서 여기 둔다. */}
      <PageHeader title={t.title} className="hidden pb-2 md:flex" />
      <div className="flex items-center gap-2">
        <SegmentedTabs
          aria-label={t.tabsLabel}
          items={SEGMENTS.map((value) => ({
            value,
            label: value === "lectures" && due > 0 ? (
              <span className="inline-flex items-center gap-1.5">
                {t[value]}
                <span aria-label={t.dueBadge(due)} className="min-w-5 rounded-full bg-accent px-1.5 text-xs font-medium leading-5 text-accent-on tabular-nums">{due}</span>
              </span>
            ) : t[value],
          }))}
          value={segment}
          onChange={select}
          hrefOf={(value) => (value === segment ? HREFS[value] : tabHref(value))}
          linkAs={Link}
          fill
          className="min-w-0 flex-1 md:max-w-sm md:flex-none"
        />
      </div>
    </div>
  );
}
