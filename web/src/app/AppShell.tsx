import { useEffect } from "react";
import { Outlet, useLocation } from "react-router-dom";
import { AppFrame } from "@forgen-team/ui";
import { useTrackInAppNavigation } from "@/src/shared/lib/in-app-history";
import { useOnline } from "@/src/shared/lib/useOnline";
import { useSectionTabs } from "@/src/widgets/punisher/nav";
import { PunisherLogo } from "@/src/shared/ui/brand";
import { LocaleSync } from "@/src/entities/punisher";
import { useT } from "@/src/shared/i18n";
import { AppPage, useFrameActions, useFrameTitle } from "./AppPage";
import { useAppCommands } from "./commands";
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
 * 단독판 앱 틀(policy/20 §2 단독판 열) = ui AppFrame: 상단 바(브랜드 · ⌘K 명령) → 섹션 탭 줄 [오늘·지도·기록] → 본문.
 * 본문은 Page(← · 제목 · 페이지 동작 · Claude 배너). 설정은 페이지 동작 "punisher 설정"으로만 들어간다.
 * ⌘K 는 셸과 같은 명령 목록(app/commands)의 명령 팔레트. 틀 폭은 오늘 화면만 today(1440), 본문 폭 상한은 Page width(app/frame).
 * LocaleSync 가 학습 설정의 언어를 화면 언어로 맞춘다.
 */
export function AppShell() {
  const t = useT(messages);
  useDocumentTitle();
  useTrackInAppNavigation();
  const { pathname } = useLocation();
  const actions = useFrameActions();
  const commands = useAppCommands();
  const tabs = useSectionTabs();
  const online = useOnline();

  return (
    <div onClickCapture={tabs.onClickCapture}>
      <LocaleSync />
      <AppFrame
        brand={{ name: t.appName, href: "/", logo: <PunisherLogo size={20} /> }}
        sections={tabs.sections}
        current={tabs.current}
        sectionsLabel={tabs.label}
        commands={commands}
        width={pathname === "/" ? "today" : "default"}
      >
        <div className="flex flex-1 flex-col gap-4">
          {/* 오프라인 읽기(sw.js): 저장해 둔 화면만 읽을 수 있고 채점·저장은 연결된 뒤에 */}
          {online ? null : <p role="status" className="border border-line bg-warning-subtle px-3 py-2 text-sm text-warning">{t.offline}</p>}
          <AppPage actions={actions.length > 0 ? <PageActions actions={actions} /> : undefined}>
            <Outlet />
          </AppPage>
        </div>
      </AppFrame>
    </div>
  );
}
