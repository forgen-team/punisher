import type { MouseEvent, ReactNode } from "react";
import { BookOpen, Map as MapIcon, Plus, RefreshCw, Repeat, ScrollText, Settings, Target, type LucideIcon } from "lucide-react";
import { Button, IconButton, Menu, useIsSplit, type MenuItemDef } from "@forgen-team/ui";
import { Link } from "@/src/shared/lib/router";
import type { PageAction } from "@/src/shared/lib/screen-chrome";
import { useT } from "@/src/shared/i18n";
import { messages } from "./i18n";
import { layoutActions } from "./actionLayout";

/** 셸 계약 아이콘 이름(lucide kebab) → 부품. 화면이 쓰는 것만 둔다 — 모르는 이름은 아이콘 없음(셸과 같다). */
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

function ActionIcon({ name }: { name?: string }) {
  const Icon = name ? ICONS[name] : undefined;
  return Icon ? <Icon className="h-4 w-4" aria-hidden="true" /> : null;
}

/** href 는 진짜 링크(새 탭 가능), onSelect 가 있으면 보통 클릭은 onSelect(셸 계약 §3). */
function runOnPlainClick(action: PageAction) {
  return (event: MouseEvent<HTMLAnchorElement>) => {
    if (!action.onSelect || event.button !== 0 || event.metaKey || event.ctrlKey || event.shiftKey || event.altKey) { return; }
    event.preventDefault();
    action.onSelect();
  };
}

function ActionButton({ action, label, variant, iconOnly }: { action: PageAction; label: string; variant: "outline" | "ghost"; iconOnly?: boolean }) {
  const icon = <ActionIcon name={action.icon} />;
  if (iconOnly) {
    return action.href ? (
      // IconButton 은 자식을 아이콘으로 덮어써 링크를 못 감싼다 — 같은 모양의 ghost 버튼 링크로.
      <Button asChild variant="ghost">
        <Link href={action.href} aria-label={action.label} title={action.label} onClick={runOnPlainClick(action)}>{icon}</Link>
      </Button>
    ) : (
      <IconButton aria-label={action.label} icon={icon} variant="ghost" onClick={action.onSelect} />
    );
  }
  const body: ReactNode = <>{icon}{label}</>;
  return action.href ? (
    <Button asChild variant={variant}>
      <Link href={action.href} onClick={runOnPlainClick(action)}>{body}</Link>
    </Button>
  ) : (
    <Button variant={variant} onClick={action.onSelect}>{body}</Button>
  );
}

function menuItem(action: PageAction): MenuItemDef {
  // 실행 동작이 있으면 버튼(onSelect), 이동만이면 진짜 링크(href).
  return action.onSelect
    ? { key: action.id, label: action.label, onSelect: action.onSelect, danger: action.danger }
    : { key: action.id, label: action.label, href: action.href, danger: action.danger };
}

/**
 * 단독판 페이지 동작(Page actions 자리) — 화면이 usePageActions 로 준 목록을 셸과 같은 규칙(app/actionLayout)으로 그린다.
 * 데스크톱(≥1024): 글자 버튼 ≤2 + "설정" + ⋯ / 폰: 아이콘 하나 + ⋯. 위험 동작의 자리(맨 아래·구분선 뒤·빨간 글자)는 ui Menu 가 정한다.
 */
export function PageActions({ actions }: { actions: PageAction[] }) {
  const t = useT(messages);
  const desktop = useIsSplit();
  if (actions.length === 0) { return null; }
  const { buttons, settings, menu } = layoutActions(actions, desktop);
  return (
    <>
      {buttons.map((action) => (
        <ActionButton key={action.id} action={action} label={action.label} variant="outline" iconOnly={!desktop} />
      ))}
      {settings ? <ActionButton action={settings} label={t.settingsShort} variant="ghost" /> : null}
      {menu.length > 0 ? <Menu label={t.moreActions} items={menu.map((action) => menuItem(action))} /> : null}
    </>
  );
}
