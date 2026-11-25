import { Suspense, useEffect, useMemo, useRef } from "react";
import { Outlet, useLocation } from "react-router-dom";
import { LocaleSync } from "@/src/entities/punisher";
import { shellHasActions, witemsRuntime, type ShellAction } from "@/src/shared/config/runtime";
import { useT } from "@/src/shared/i18n";
import { useTrackInAppNavigation } from "@/src/shared/lib/in-app-history";
import { actionsSignature, useScreenChrome } from "@/src/shared/lib/screen-chrome";
import { useSearchShortcut } from "@/src/shared/lib/shortcuts";
import { useRouter } from "@/src/shared/lib/router";
import { FeedbackHost, ListSkeleton } from "@/src/shared/ui/kit";
import { ClaudeStatusBanner } from "@/src/widgets/claude-status";
import { LearnNav } from "@/src/widgets/punisher/nav";
import { messages } from "@/src/app/i18n";
import { titleKeyFor } from "@/src/app/titles";

/**
 * 셸 안에서의 앱 틀: Witems 가 상단 바·메뉴·여백·제목(데스크톱 h1)을 맡으므로 학습 탭과 본문만 둔다.
 * 제목: 화면이 정한 구체적인 제목(강의 제목 등, useScreenTitle) → 없으면 경로별 제목(오늘은 null = manifest "학습").
 * 페이지 동작: 셸이 ctx.setActions 를 가지면 화면의 동작 + 마지막에 '학습 설정'을 넘긴다(UI-PAGE-LAYOUT §1).
 * 구버전 셸이면 넘기지 않고, LearnNav 가 탭 옆에 설정 링크를, 강의 화면은 ⋯ 메뉴에 이동 항목을 그대로 둔다.
 */
export function WitemsShell() {
  const { pathname } = useLocation();
  const t = useT(messages);
  const chrome = useScreenChrome();
  const router = useRouter();
  useTrackInAppNavigation();
  useSearchShortcut(() => router.push("/search"));

  const pathTitle = useMemo(() => {
    const key = titleKeyFor(pathname);
    return key === "titleToday" ? null : t[key];
  }, [pathname, t]);
  const title = chrome.title ?? pathTitle;
  useEffect(() => {
    witemsRuntime()?.setTitle(title);
  }, [title]);

  const actions = useMemo<ShellAction[]>(() => {
    const settings: ShellAction = { id: "settings", label: t.learnSettings, icon: "settings", href: "/settings", kind: "settings" };
    const search: ShellAction = { id: "search", label: t.search, icon: "search", href: "/search" };
    const own = chrome.actions ?? [];
    const withSearch = pathname.startsWith("/search") ? own : [...own, search];
    return pathname.startsWith("/settings") ? withSearch : [...withSearch, settings];
  }, [chrome.actions, pathname, t.learnSettings, t.search]);
  const sent = useRef("");
  useEffect(() => {
    if (!shellHasActions()) { return; }
    const signature = actionsSignature(actions);
    if (signature === sent.current) { return; }
    sent.current = signature;
    witemsRuntime()?.setActions?.(actions);
  }, [actions]);

  return (
    <div className="flex min-w-0 flex-col gap-4">
      <LocaleSync />
      <ClaudeStatusBanner />
      <LearnNav />
      <div className="flex min-w-0 flex-1 flex-col">
        <Suspense fallback={<ListSkeleton rows={6} />}>
          <Outlet />
        </Suspense>
      </div>
      <FeedbackHost />
    </div>
  );
}
