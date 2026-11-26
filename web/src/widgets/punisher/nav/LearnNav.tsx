import { useEffect } from "react";
import { Search, Settings } from "lucide-react";
import { isWitems, shellHasActions, shellHasSubNav, witemsRuntime } from "@/src/shared/config/runtime";
import { Link, useRouter } from "@/src/shared/lib/router";
import { PageHeader, SegmentedTabs, cn, focusRing } from "@/src/shared/ui/kit";
import { useT } from "@/src/shared/i18n";
import { useLocation } from "react-router-dom";
import { HREFS, SEGMENTS, rememberTab, segmentOf, tabHref, tabScroll, type LearnSegment } from "./lib/tabMemory";
import { useActivity } from "@/src/entities/punisher";
import { messages } from "./i18n";

/**
 * 학습 화면 상단: 제목(단독판 데스크톱만) + 탭 [강의·지도·기록] — 탭 줄에는 탭만 둔다(UI-PAGE-LAYOUT §2).
 * 탭은 링크(⌘/Ctrl·가운데 클릭 새 탭), 화살표는 포커스만. 다른 탭에 갔다 돌아오면 그 탭의 마지막 주소·스크롤로,
 * 지금 탭을 다시 누르면 처음 화면으로.
 * 학습 설정: Witems 셸이 페이지 동작(ctx.setActions)을 그리면 거기(⋯ 마지막 '학습 설정')로 옮기고, 구버전 셸이면
 * 지금처럼 탭 옆 링크로 둔다. 단독판은 셸 상단 바의 설정.
 * 강의 뷰어·즉석 강의·설정 화면에서는 자체 상단 바를 쓰므로 렌더하지 않는다.
 * 사이드바 하위 메뉴를 그리는 Witems 셸(manifest nav.items — ctx.setBadges 가 있음)에서는 탭이 그 메뉴와 겹치므로
 * 그리지 않고, 복습 수는 셸의 '강의' 항목 숫자(setBadges {"#/": n})로 보낸다. 탭 이동·기억은 witems/RouteSync 가 맡는다.
 */
export function LearnNav() {
  const { pathname, search } = useLocation();
  const router = useRouter();
  const segment = segmentOf(pathname);
  const t = useT(messages);
  // 앱 안 배지(W5a): 지금 볼 복습·카드 수 — Witems 에는 앱이 보낼 알림 경로가 없어 이것으로 알린다(쉬는 날은 숨김).
  const activity = useActivity().data;
  const due = activity && !activity.restToday ? activity.today.reviewsDue + activity.today.cardsDue : 0;
  useEffect(() => {
    if (segment) { rememberTab(segment, `${pathname}${search}`); }
  }, [segment, pathname, search]);
  const subNavInShell = shellHasSubNav();
  useEffect(() => {
    if (subNavInShell) { witemsRuntime()?.setBadges?.({ "#/": due }); }
  }, [subNavInShell, due]);
  if (!segment) { return null; }
  const inWitems = isWitems();
  const settingsInTabRow = inWitems && !shellHasActions();
  // 셸이 하위 메뉴를 그리면 탭 줄은 없다 — 구버전 셸(setActions 없음)이면 검색·설정 링크만 남긴다.
  if (subNavInShell && !settingsInTabRow) { return null; }

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
      {/* 단독판 셸 상단 바는 화면 제목을 보이지 않으므로 md 이상에서 여기 둔다(witems 는 셸이 그림 — docs/SYNC.md). */}
      {inWitems ? null : <PageHeader title={t.title} className="hidden pb-2 md:flex" />}
      <div className="flex items-center gap-2">
        {subNavInShell ? <div className="flex-1" /> : <SegmentedTabs
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
        />}
        {settingsInTabRow ? (
          <Link href="/search" aria-label={t.search} className={cn("inline-flex min-h-touch shrink-0 items-center rounded-md px-2 text-fg-secondary hover:bg-canvas-hover hover:text-fg", focusRing)}>
            <Search className="h-4 w-4" aria-hidden="true" />
          </Link>
        ) : null}
        {settingsInTabRow ? (
          <Link
            href="/settings"
            className={cn("inline-flex min-h-touch shrink-0 items-center gap-1.5 rounded-md px-2 text-sm text-fg-secondary hover:bg-canvas-hover hover:text-fg", focusRing)}
          >
            <Settings className="h-4 w-4" aria-hidden="true" />
            {/* 폰에서도 글자를 둔다(톱니만으로는 무엇인지 몰랐다 — P2·점검 30). */}
            <span>{t.settings}</span>
          </Link>
        ) : null}
      </div>
    </div>
  );
}
