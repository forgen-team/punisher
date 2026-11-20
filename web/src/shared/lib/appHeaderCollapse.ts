import { useEffect } from "react";

// 휴대폰 강의 화면의 고정 바를 하나로(⑤ 모바일): 아래로 읽어 내려가면 셸 상단 바([data-app-header])를 위로 숨기고,
// 위로 되돌아가면 다시 보인다. 강의 바([data-lecture-topbar])는 늘 보이고 --app-header-offset 을 따라 맨 위로 붙는다.
// 숨김 표시는 <html data-app-header-state="hidden"> — 셸 바의 [data-app-header] 와 이름을 나눠 그 선택자가 바에만 걸리게 한다.
// 실제 이동·offset 변경은 globals.css 가 휴대폰 폭에서만 하고, 바 안에 포커스가 있으면 숨기지 않는다.

/** 휴대폰 폭(Tailwind fold 600px 미만 — useIsSplitLayout 과 같은 경계). */
export const PHONE_QUERY = "(max-width: 599.98px)";
/** 이 스크롤 위치 위(셸 바 + 강의 바 높이 정도)에서는 늘 보인다. */
export const HEADER_SHOW_ZONE = 96;
/** 방향이 바뀐 뒤 이만큼 더 움직여야 숨기거나 보인다(손가락 떨림·관성 끝의 작은 되튐 무시). */
export const HEADER_TOGGLE_DELTA = 12;

export interface HeaderScrollState {
  hidden: boolean;
  /** 지금 방향의 기준점 — 보이는 동안은 가장 위, 숨긴 동안은 가장 아래 위치 */
  anchorY: number;
}

/** 스크롤 위치 y 를 받아 다음 상태를 정한다(순수 함수). */
export function nextHeaderState(
  state: HeaderScrollState,
  y: number,
  showZone = HEADER_SHOW_ZONE,
  delta = HEADER_TOGGLE_DELTA,
): HeaderScrollState {
  if (y <= showZone) { return { hidden: false, anchorY: y }; }
  if (state.hidden) {
    if (state.anchorY - y > delta) { return { hidden: false, anchorY: y }; }
    return y > state.anchorY ? { hidden: true, anchorY: y } : state;
  }
  if (y - state.anchorY > delta) { return { hidden: true, anchorY: y }; }
  return y < state.anchorY ? { hidden: false, anchorY: y } : state;
}

let frozenUntil = 0;

/**
 * 코드가 일으킨 스크롤(구절을 강의 바 바로 아래로 올리기 등) 동안 숨김 상태를 바꾸지 않는다 —
 * 스크롤 도중 셸 바가 나타나거나 사라지면 강의 바 위치가 바뀌어 맞춘 자리가 어긋난다.
 */
export function freezeAppHeader(ms = 800): void {
  frozenUntil = Date.now() + ms;
}

/** <html> 에 다는 숨김 상태 속성(data-app-header-state). */
export const APP_HEADER_STATE_ATTR = "data-app-header-state";

/** 셸 바 숨김 표시를 <html> 에 달거나 뗀다. */
export function setAppHeaderHidden(hidden: boolean): void {
  const root = document.documentElement;
  if (hidden) { root.setAttribute(APP_HEADER_STATE_ATTR, "hidden"); } else { root.removeAttribute(APP_HEADER_STATE_ATTR); }
}

/** 강의 바가 있는 화면에서 부른다. 화면을 떠나면 셸 바를 다시 보인다. */
export function useCollapsingAppHeader(): void {
  useEffect(() => {
    const phone = window.matchMedia(PHONE_QUERY);
    let state: HeaderScrollState = { hidden: false, anchorY: window.scrollY };
    let frame = 0;
    const apply = (next: HeaderScrollState) => {
      if (next.hidden !== state.hidden) { setAppHeaderHidden(next.hidden); }
      state = next;
    };
    const update = () => {
      frame = 0;
      const y = window.scrollY;
      if (!phone.matches) {
        apply({ hidden: false, anchorY: y });
        return;
      }
      // 시트가 body 스크롤을 잠그면(position: fixed) scrollY 가 0 이 된다 — 그동안은 그대로 둔다.
      if (document.body.style.position === "fixed") { return; }
      if (Date.now() < frozenUntil) {
        state = { ...state, anchorY: y };
        return;
      }
      apply(nextHeaderState(state, y));
    };
    const onScroll = () => {
      if (!frame) { frame = window.requestAnimationFrame(update); }
    };
    window.addEventListener("scroll", onScroll, { passive: true });
    phone.addEventListener("change", update);
    return () => {
      window.removeEventListener("scroll", onScroll);
      phone.removeEventListener("change", update);
      if (frame) { window.cancelAnimationFrame(frame); }
      setAppHeaderHidden(false);
    };
  }, []);
}
