import { Suspense, useEffect, useMemo, useRef } from "react";
import { Outlet, useLocation } from "react-router-dom";
import { LocaleSync } from "@/src/entities/punisher";
import { shellHasActions, shellHasCommands, witemsRuntime, type ShellAction } from "@/src/shared/config/runtime";
import { lectureHref, useStartReview, useToday } from "@/src/entities/punisher";
import { checkHref } from "@/src/features/punisher/check-start";
import { describePunisherError } from "@/src/shared/api/punisher";
import { recommendationHref } from "@/src/widgets/punisher/today";
import { buildCommands } from "./commands";
import { useT } from "@/src/shared/i18n";
import { useTrackInAppNavigation } from "@/src/shared/lib/in-app-history";
import { actionsSignature, useScreenChrome } from "@/src/shared/lib/screen-chrome";
import { useSearchShortcut } from "@/src/shared/lib/shortcuts";
import { useRouter } from "@/src/shared/lib/router";
import { ListSkeleton, useToast } from "@forgen-team/ui";
import { ClaudeStatusBanner } from "@/src/widgets/claude-status";
import { messages } from "@/src/app/i18n";
import { titleKeyFor } from "@/src/app/titles";
import { useSectionBadges } from "./sectionBadges";

/**
 * 셸 안에서의 앱 틀: Witems 가 상단 바·메뉴·여백·제목(데스크톱 h1)·← 를 맡으므로 본문만 둔다. 학습 섹션
 * [강의·지도·기록]은 셸 섹션 메뉴(manifest nav.items, 경로 — D8)가 그린다(본문 탭 줄·LearnNav 없음). 복습 수는
 * 그 메뉴의 '강의' 숫자로 보낸다(useSectionBadges).
 * 제목: 화면이 정한 구체적인 제목(강의 제목 등, useScreenTitle) → 없으면 경로별 제목(오늘은 null = manifest "학습").
 * 페이지 동작: 셸이 ctx.setActions 를 가지면 화면의 동작 + 마지막에 '학습 설정'을 넘긴다(UI-PAGE-LAYOUT §1).
 * 구버전 셸이면 넘기지 않고, 강의 화면은 ⋯ 메뉴에 이동 항목을 그대로 둔다.
 */
export function WitemsShell() {
  const { pathname } = useLocation();
  const t = useT(messages);
  const chrome = useScreenChrome();
  const router = useRouter();
  const toast = useToast();
  useTrackInAppNavigation();
  useSectionBadges();
  // ⌘K: 새 셸은 캡처 단계에서 통합 검색을 연다(명령은 setCommands 로 넣는다) — 그때는 앱이 키를 걸지 않는다.
  useSearchShortcut(() => router.push("/search"), !shellHasCommands());

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

  // ⌘K 명령(APP-CONTRACT §3 "명령"): 셸이 setCommands 를 가지면 학습 명령을 넣는다. 구버전 셸이면 아무것도 하지 않는다.
  const today = useToday();
  const review = useStartReview();
  const reviewRef = useRef(review);
  reviewRef.current = review;
  const next = (today.data?.items ?? []).find((item) => item.kind !== "resume");
  const nextTitle = next ? next.conceptName || next.title : "";
  const nextHref = next ? (next.lectureId ? lectureHref(next.lectureId) : recommendationHref(next.conceptId)) : "";
  const commands = useMemo(
    () => buildCommands(t, {
      next: next ? { title: nextTitle, href: nextHref } : undefined,
      startReview: () => reviewRef.current.mutate(undefined, {
        onSuccess: (detail) => router.push(checkHref(detail.session.id)),
        onError: (err) => toast.show(describePunisherError(err) || t.noReviews, { tone: "danger" }),
      }),
    }),
    [t, next, nextTitle, nextHref, router, toast],
  );
  const sentCommands = useRef("");
  useEffect(() => {
    if (!shellHasCommands()) { return; }
    const signature = JSON.stringify(commands.map(({ onSelect, ...rest }) => ({ ...rest, run: Boolean(onSelect) })));
    if (signature === sentCommands.current) { return; }
    sentCommands.current = signature;
    witemsRuntime()?.setCommands?.(commands);
  }, [commands]);

  return (
    <div className="flex min-w-0 flex-col gap-4">
      <LocaleSync />
      <ClaudeStatusBanner />
      <div className="flex min-w-0 flex-1 flex-col">
        <Suspense fallback={<ListSkeleton rows={6} />}>
          <Outlet />
        </Suspense>
      </div>
    </div>
  );
}
