import { Suspense, useEffect } from "react";
import { Outlet, useLocation } from "react-router-dom";
import { useTrackInAppNavigation } from "@/src/shared/lib/in-app-history";
import { Settings } from "lucide-react";
import { Link } from "@/src/shared/lib/router";
import { ListSkeleton, cn, focusRing } from "@/src/shared/ui/kit";
import { LearnNav } from "@/src/widgets/punisher/nav";
import { ClaudeStatusBanner } from "@/src/widgets/claude-status";
import { BrandMark } from "@/src/shared/ui/brand";

const TITLES: Array<[RegExp, string]> = [
  [/^\/map/, "지식 지도"],
  [/^\/history/, "학습 기록"],
  [/^\/chapters\/new/, "다음 챕터 고르기"],
  [/^\/lectures\//, "강의"],
  [/^\/inquiries\//, "질문 해설"],
  [/^\/instant\//, "즉석 강의 생성 중"],
  [/^\/settings\/prompts/, "강의 프롬프트"],
  [/^\/settings/, "설정"],
];

function useDocumentTitle(pathname: string) {
  useEffect(() => {
    const match = TITLES.find(([re]) => re.test(pathname));
    document.title = match ? `${match[1]} · Punisher` : "오늘의 강의 · Punisher";
  }, [pathname]);
}

/**
 * 앱 셸: 고정 상단 바(로고 · 설정) + Claude 상태 배너 + 학습 탭(강의·지도·기록) + 본문.
 * 폭·좌우 여백은 witems 학습 레이아웃과 같다(최대 1600px, px-4 → fold 6 → lg 8). 본문은 화면별로 가독 폭을 둔다.
 * 강의 뷰어·즉석 강의·설정 화면은 LearnNav 가 스스로 숨는다.
 */
export function AppShell() {
  const { pathname } = useLocation();
  useDocumentTitle(pathname);
  useTrackInAppNavigation(pathname);
  const onSettings = pathname.startsWith("/settings");

  return (
    <div className="flex min-h-screen flex-col">
      <header className="sticky top-0 z-nav border-b border-line bg-canvas-translucent pt-safe-top backdrop-blur">
        <div className="mx-auto flex h-12 w-full max-w-[1600px] items-center justify-between px-4 fold:px-6 lg:px-8">
          <Link href="/" aria-label="Punisher 홈" className={cn("flex items-center rounded-md", focusRing)}>
            <BrandMark />
          </Link>
          <Link
            href="/settings"
            aria-label="설정"
            aria-current={onSettings ? "page" : undefined}
            className={cn(
              "inline-flex min-h-touch items-center gap-1.5 rounded-md px-2 text-sm text-fg-secondary hover:bg-canvas-hover hover:text-fg",
              onSettings && "text-fg",
              focusRing,
            )}
          >
            <Settings className="h-4 w-4" aria-hidden="true" />
            <span className="hidden sm:inline">설정</span>
          </Link>
        </div>
      </header>
      <main className="mx-auto flex w-full max-w-[1600px] flex-1 flex-col gap-4 px-4 pb-[calc(3rem+var(--safe-bottom))] pt-3 fold:px-6 lg:px-8">
        <ClaudeStatusBanner />
        <LearnNav />
        <div className="flex min-w-0 flex-1 flex-col">
          <Suspense fallback={<ListSkeleton rows={6} />}>
            <Outlet />
          </Suspense>
        </div>
      </main>
    </div>
  );
}
