import { Suspense, useMemo, type MouseEvent, type ReactNode } from "react";
import { useLocation } from "react-router-dom";
import { ListSkeleton, Page, useUI } from "@forgen-team/ui";
import { canGoBack } from "@/src/shared/lib/in-app-history";
import { useRouter } from "@/src/shared/lib/router";
import { useScreenChrome, type PageAction } from "@/src/shared/lib/screen-chrome";
import { useT } from "@/src/shared/i18n";
import { ClaudeStatusBanner, useClaudeStatusIssue } from "@/src/widgets/claude-status/ClaudeStatusBanner";
import { backFallbackFor, pageWidthFor, usesGeneration } from "./frame";
import { messages } from "./i18n";
import { withAppSettings } from "./actionLayout";
import { titleKeyFor } from "./titles";

/** 화면 제목 = 화면이 준 구체적인 제목(강의 제목 등) → 없으면 경로별 제목(오늘은 "오늘", D11 ①). */
export function useFrameTitle(): string {
  const { pathname } = useLocation();
  const t = useT(messages);
  const { title } = useScreenChrome();
  return title ?? t[titleKeyFor(pathname)];
}

/** 이 화면의 페이지 동작 = 화면이 준 것 + 끝에 "punisher 설정"(/settings 에서는 뺌). */
export function useFrameActions(): PageAction[] {
  const { pathname } = useLocation();
  const t = useT(messages);
  const { actions } = useScreenChrome();
  return useMemo(() => withAppSettings(actions, pathname, t.appSettings), [actions, pathname, t.appSettings]);
}

function isPlainClick(event: MouseEvent): boolean {
  return event.button === 0 && !event.metaKey && !event.ctrlKey && !event.shiftKey && !event.altKey;
}

/**
 * 화면 틀(@forgen-team/ui Page) — 두 셸이 <Outlet/> 을 이것으로 감싼다. 폭은 경로표(app/frame), 생성을 쓰는 화면이면
 * 머리 바로 아래 Claude 상태 배너. 셸 안(UIProvider inShell)이면 Page 가 머리를 그리지 않고 폭·배너만 둔다.
 * 단독판: h1 = 화면 제목(+ 화면이 준 부제 — useScreenSubtitle), 섹션 첫 화면이 아니면 ← (앱 안에서 왔으면 뒤로, 아니면 그 섹션 첫 화면으로 replace),
 * 동작 자리 = actions(AppShell 이 PageActions 로 그린 것 — 배치는 ui PageActions).
 */
export function AppPage({ actions, children }: { actions?: ReactNode; children: ReactNode }) {
  const { pathname } = useLocation();
  const { inShell } = useUI();
  const router = useRouter();
  const title = useFrameTitle();
  const { subtitle } = useScreenChrome();
  const issue = useClaudeStatusIssue();
  const fallback = backFallbackFor(pathname);

  // Page 의 ← 는 진짜 링크(href = 바로 들어왔을 때의 목적지 — 새 탭도 그곳). 보통 클릭만 받아 목적지를 정한다.
  function handleClickCapture(event: MouseEvent<HTMLDivElement>) {
    if (!fallback || !isPlainClick(event)) { return; }
    const back = event.target instanceof Element ? event.target.closest(".fg-page__back") : null;
    if (!back) { return; }
    event.preventDefault();
    event.stopPropagation();
    if (canGoBack()) { router.back(); } else { router.replace(fallback); }
  }

  return (
    <div className="flex min-w-0 flex-1 flex-col" onClickCapture={inShell ? undefined : handleClickCapture}>
      <Page
        title={title}
        subtitle={subtitle ?? undefined}
        back={fallback ? { href: fallback } : undefined}
        actions={actions}
        width={pageWidthFor(pathname)}
        banner={issue && usesGeneration(pathname) ? <ClaudeStatusBanner /> : undefined}
      >
        <Suspense fallback={<ListSkeleton rows={6} />}>{children}</Suspense>
      </Page>
    </div>
  );
}
