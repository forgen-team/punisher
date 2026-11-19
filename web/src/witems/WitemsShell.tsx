import { Suspense, useEffect } from "react";
import { Outlet, useLocation } from "react-router-dom";
import { LocaleSync } from "@/src/entities/punisher";
import { witemsRuntime } from "@/src/shared/config/runtime";
import { useT } from "@/src/shared/i18n";
import { useTrackInAppNavigation } from "@/src/shared/lib/in-app-history";
import { ListSkeleton } from "@/src/shared/ui/kit";
import { ClaudeStatusBanner } from "@/src/widgets/claude-status";
import { LearnNav } from "@/src/widgets/punisher/nav";
import { messages } from "@/src/app/i18n";
import { titleKeyFor } from "@/src/app/titles";

/**
 * 셸 안에서의 앱 틀: Witems 가 상단 바·메뉴·여백·제목(데스크톱 h1)을 맡으므로 학습 탭과 본문만 둔다.
 * 하위 화면 제목은 셸 상단 바로 보낸다(오늘은 null = manifest 제목 "학습").
 */
export function WitemsShell() {
  const { pathname } = useLocation();
  const t = useT(messages);
  useTrackInAppNavigation(pathname);

  useEffect(() => {
    const key = titleKeyFor(pathname);
    witemsRuntime()?.setTitle(key === "titleToday" ? null : t[key]);
  }, [pathname, t]);

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
    </div>
  );
}
