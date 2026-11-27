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
