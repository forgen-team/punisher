import { useEffect } from "react";
import { Outlet } from "react-router-dom";
import { useTrackInAppNavigation } from "@/src/shared/lib/in-app-history";
import { useSearchShortcut } from "@/src/shared/lib/shortcuts";
import { Link, useRouter } from "@/src/shared/lib/router";
import { cn, focusRing } from "@/src/shared/lib/cn";
import { useOnline } from "@/src/shared/lib/useOnline";
import { LearnNav } from "@/src/widgets/punisher/nav";
import { BrandMark } from "@/src/shared/ui/brand";
import { LocaleSync } from "@/src/entities/punisher";
import { useT } from "@/src/shared/i18n";
import { AppPage, useFrameActions, useFrameTitle } from "./AppPage";
import { messages } from "./i18n";
import { PageActions } from "./PageActions";

/** 문서 제목: "화면 제목 · punisher". */
function useDocumentTitle() {
  const t = useT(messages);
  const title = useFrameTitle();
  useEffect(() => {
    document.title = `${title} · ${t.appName}`;
  }, [t.appName, title]);
}

/**
 * 단독판 앱 틀(policy/20 §2 단독판 열): 고정 머리 = 앱 이름 줄(브랜드만) + 섹션 탭 줄 [오늘·지도·기록],
 * 그 아래 Page(← · 제목 · 페이지 동작 · Claude 배너 · 본문). 설정은 페이지 동작 "punisher 설정"으로만 들어간다.
 * ⌘K: 명령 팔레트가 없어 기록(찾기)으로 보낸다. LocaleSync 가 학습 설정의 언어를 화면 언어로 맞춘다.
 * 폭·좌우 여백은 witems 학습 레이아웃과 같다(최대 1600px, px-4 → fold 6 → lg 8). 본문 폭 상한은 Page width(app/frame).
 */
export function AppShell() {
  const t = useT(messages);
  useDocumentTitle();
  useTrackInAppNavigation();
  const router = useRouter();
  useSearchShortcut(() => router.push("/history"));
  const actions = useFrameActions();
  const online = useOnline();

  return (
    // overflow-x-clip: 넓은 코드·표는 제 상자 안에서 스크롤하고 화면 전체가 옆으로 밀리지는 않게(clip 은 sticky 를 깨지 않는다).
    <div className="flex min-h-screen flex-col overflow-x-clip">
      <LocaleSync />
      {/* data-app-header: 휴대폰 강의 화면에서 아래로 읽는 동안 숨긴다 — 상태는 <html data-app-header-state>(shared/lib/appHeaderCollapse · globals.css).
          높이 = 앱 이름 줄 48 + 탭 줄 44 = --app-topbar-h. */}
      <header data-app-header="" className="sticky top-0 z-nav bg-canvas-translucent pt-safe-top backdrop-blur">
        <div className="mx-auto flex h-12 w-full max-w-[1600px] items-center px-4 fold:px-6 lg:px-8">
          <Link href="/" aria-label={t.home} className={cn("flex items-center", focusRing)}>
            <BrandMark />
          </Link>
        </div>
        <div className="mx-auto flex h-11 w-full max-w-[1600px] flex-col justify-end px-4 fold:px-6 lg:px-8">
          <LearnNav />
        </div>
      </header>
      <main className="mx-auto flex w-full max-w-[1600px] flex-1 flex-col gap-4 px-4 pb-[calc(3rem+var(--safe-bottom))] pt-4 fold:px-6 lg:px-8">
        {/* 오프라인 읽기(sw.js): 저장해 둔 화면만 읽을 수 있고 채점·저장은 연결된 뒤에 */}
        {online ? null : <p role="status" className="border border-line bg-warning-subtle px-3 py-2 text-sm text-warning">{t.offline}</p>}
        <AppPage actions={actions.length > 0 ? <PageActions actions={actions} /> : undefined}>
          <Outlet />
        </AppPage>
      </main>
    </div>
  );
}
