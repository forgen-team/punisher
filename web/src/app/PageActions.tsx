import { BookOpen, Map as MapIcon, Plus, RefreshCw, Repeat, ScrollText, Settings, Target, type LucideIcon } from "lucide-react";
import { PageActions as UIPageActions, type AppAction } from "@forgen-team/ui";
import type { PageAction } from "@/src/shared/lib/screen-chrome";
import { useT } from "@/src/shared/i18n";
import { messages } from "./i18n";

/**
 * 셸 계약 아이콘 이름(lucide kebab, ctx.setActions 로 넘기는 값) ↔ 단독판에 그릴 그림. 화면이 쓰는 것만 둔다 —
 * 모르는 이름은 아이콘 없음(셸과 같다).
 */
const ICONS: Record<string, LucideIcon> = {
  plus: Plus,
  settings: Settings,
  map: MapIcon,
  "scroll-text": ScrollText,
  "book-open": BookOpen,
  repeat: Repeat,
  "refresh-cw": RefreshCw,
  target: Target,
};

/**
 * 화면이 준 동작(셸 계약 모양: icon = 이름, kind 없음 = ⋯) → ui AppAction(icon = 그림, kind "more" = ⋯).
 * 배치(데스크톱 글자 버튼 ≤2 + 설정 + ⋯ / 폰 아이콘 하나 + ⋯, 위험은 ⋯ 맨 아래)는 ui PageActions 가 셸과 같은 규칙으로 정한다.
 */
export function toAppActions(actions: readonly PageAction[]): AppAction[] {
  return actions.map(({ icon, kind, ...rest }) => {
    const Icon = icon ? ICONS[icon] : undefined;
    return { ...rest, kind: kind ?? "more", icon: Icon ? <Icon className="h-4 w-4" aria-hidden="true" /> : undefined };
  });
}

/** 단독판 페이지 동작(Page actions 자리). 셸 안에서는 같은 목록을 ctx.setActions 로 넘긴다(WitemsShell). */
export function PageActions({ actions }: { actions: PageAction[] }) {
  const t = useT(messages);
  if (actions.length === 0) { return null; }
  return <UIPageActions actions={toAppActions(actions)} moreLabel={t.moreActions} />;
}
