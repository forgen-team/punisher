import { useEffect, useRef } from "react";
import { Outlet } from "react-router-dom";
import { LocaleSync } from "@/src/entities/punisher";
import { shellHasActions, shellHasCommands, witemsRuntime } from "@/src/shared/config/runtime";
import { useTrackInAppNavigation } from "@/src/shared/lib/in-app-history";
import { actionsSignature, useScreenChrome } from "@/src/shared/lib/screen-chrome";
import { useSearchShortcut } from "@/src/shared/lib/shortcuts";
import { useRouter } from "@/src/shared/lib/router";
import { AppPage, useFrameActions, useFrameTitle } from "@/src/app/AppPage";
import { useAppCommands } from "@/src/app/commands";
import { useSectionBadges } from "./sectionBadges";

/**
 * 셸 안에서의 앱 틀: Witems 가 상단 바·메뉴·제목(데스크톱 h1)·← 를 맡고, 앱은 Page(inShell)로 본문 폭·Claude 배너만 둔다.
 * 섹션 [오늘·지도·기록]은 셸 섹션 메뉴(manifest nav.items)가 그리고, '오늘' 숫자는 useSectionBadges 가 보낸다.
 * 제목: 화면이 정한 구체적인 제목(useScreenTitle) → 없으면 경로별 제목(오늘도 "오늘", D11 ①) → ctx.setTitle.
 * 페이지 동작: 화면의 동작 + 끝에 "punisher 설정"(/settings 에서는 뺌)을 ctx.setActions 로. 구버전 셸이면 넘기지 않는다.
 * ⌘K: 새 셸은 명령(setCommands — 단독판 팔레트와 같은 목록)을 받고 키는 셸이 쥔다. 앱은 팔레트를 띄우지 않는다.
 * 구버전 셸(setCommands 없음)이면 앱이 키를 걸어 기록(찾기)으로 보낸다.
 */
export function WitemsShell() {
  const router = useRouter();
  useTrackInAppNavigation();
  useSectionBadges();
  useSearchShortcut(() => router.push("/history"), !shellHasCommands());

  const title = useFrameTitle();
  useEffect(() => {
    witemsRuntime()?.setTitle(title);
  }, [title]);

  // 부제(오늘 날짜 등) — 셸 머리 제목 아래(ctx.setSubtitle, witems#54). 없는 셸이면 건너뛴다.
  const { subtitle } = useScreenChrome();
  useEffect(() => {
    witemsRuntime()?.setSubtitle?.(subtitle);
  }, [subtitle]);

  const actions = useFrameActions();
  const sent = useRef("");
  useEffect(() => {
    if (!shellHasActions()) { return; }
    const signature = actionsSignature(actions);
    if (signature === sent.current) { return; }
    sent.current = signature;
    witemsRuntime()?.setActions?.(actions);
  }, [actions]);

  // ⌘K 명령(APP-CONTRACT §3 "명령"): 셸이 setCommands 를 가지면 넣는다. 구버전 셸이면 아무것도 하지 않는다.
  const commands = useAppCommands();
  const sentCommands = useRef("");
  useEffect(() => {
    if (!shellHasCommands()) { return; }
    const signature = JSON.stringify(commands.map(({ onSelect, ...rest }) => ({ ...rest, run: Boolean(onSelect) })));
    if (signature === sentCommands.current) { return; }
    sentCommands.current = signature;
    witemsRuntime()?.setCommands?.(commands);
  }, [commands]);

  return (
    <div className="flex min-w-0 flex-col">
      <LocaleSync />
      <AppPage>
        <Outlet />
      </AppPage>
    </div>
  );
}
