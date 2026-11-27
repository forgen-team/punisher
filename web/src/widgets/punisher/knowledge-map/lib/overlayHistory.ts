import { canGoBack } from "@/src/shared/lib/in-app-history";
import type { AppRouter } from "@/src/shared/lib/router";

/**
 * 지도 시트(`?panel=`)와 개념 상세(`?node=`)는 push 로 열어서, 닫을 때 뒤로 간다(기록에 닫힌 시트가 남지 않게).
 * 주소로 바로 들어와 앞 기록이 없으면 시트 없는 주소로 replace 한다. 우리가 push·replace 로 만든 주소만 기억한다 —
 * 같은 주소가 다른 길로 열렸다면 뒤로 가지 않는다.
 */
const opened = new Set<string>();

/** 시트·상세를 연다(push). 이미 열린 것을 갈아 끼우는 거면 `replaceFrom`(지금 주소)을 주고 replace 한다. */
export function openOverlay(router: AppRouter, href: string, replaceFrom?: string): void {
  opened.add(href);
  if (replaceFrom === undefined) {
    router.push(href, { scroll: false });
    return;
  }
  if (replaceFrom !== href) { opened.delete(replaceFrom); }
  router.replace(href, { scroll: false });
}

/** 닫는다: 우리가 연 주소면 뒤로, 아니면 `closedHref`(시트 없는 주소)로 replace. */
export function closeOverlay(router: AppRouter, currentHref: string, closedHref: string): void {
  if (opened.delete(currentHref) && canGoBack()) {
    router.back();
    return;
  }
  router.replace(closedHref, { scroll: false });
}
