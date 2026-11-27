import type { PageAction } from "@/src/shared/lib/screen-chrome";

/** 앱 설정 동작의 id — 셸 계약 kind "settings" 자리. */
export const SETTINGS_ACTION_ID = "settings";

/**
 * 화면이 준 페이지 동작 끝에 "punisher 설정"을 붙인다(셸 계약 §3: 앱 설정 진입은 kind "settings").
 * 지금 그 화면(/settings)에서는 뺀다. 화면이 이미 settings 를 줬으면 그대로 둔다.
 */
export function withAppSettings(actions: PageAction[] | null, pathname: string, label: string): PageAction[] {
  const own = actions ?? [];
  if (pathname === "/settings" || own.some((a) => a.kind === "settings")) { return own; }
  return [...own, { id: SETTINGS_ACTION_ID, label, icon: "settings", href: "/settings", kind: "settings" }];
}

export interface ActionLayout {
  /** 밖에 그리는 버튼 — 데스크톱은 글자 버튼, 폰은 아이콘 하나 */
  buttons: PageAction[];
  /** 데스크톱의 설정 글자 버튼("설정") */
  settings: PageAction | null;
  /** ⋯ 안 — 위험 동작은 어디 있든 ui Menu 가 맨 아래 구분선 뒤에 둔다 */
  menu: PageAction[];
}

/**
 * 셸 계약 §3·policy/20 §6 배치 규칙(witems 셸과 같다) — 단독판도 이대로 그린다. 위험 동작은 밖에 꺼내지 않는다.
 * 데스크톱: 앞의 default 2개 글자 버튼 + "설정" + ⋯(나머지).
 * 폰: 아이콘 있는 첫 default 하나 + ⋯(나머지 → 앱 설정).
 */
export function layoutActions(actions: PageAction[], desktop: boolean): ActionLayout {
  const settings = actions.find((a) => a.kind === "settings" && !a.danger) ?? null;
  const defaults = actions.filter((a) => a.kind === "default" && !a.danger);
  const rest = actions.filter((a) => a !== settings && !defaults.includes(a));
  if (desktop) {
    return { buttons: defaults.slice(0, 2), settings, menu: [...defaults.slice(2), ...rest] };
  }
  const first = defaults.find((a) => a.icon);
  return {
    buttons: first ? [first] : [],
    settings: null,
    menu: [...defaults.filter((a) => a !== first), ...rest, ...(settings ? [settings] : [])],
  };
}
