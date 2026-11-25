import { useEffect } from "react";

/**
 * 앱 안 이동 경고 — 저장하지 않은 편집이 있는 화면이 등록해 두면, 앱 링크·router.push 가 떠나기 전에
 * 앱 안 확인 창을 띄운다(shared/lib/router). 탭 닫기·새로고침은 브라우저 beforeunload 로 묻는다.
 * 셸 메뉴·뒤로가기처럼 앱이 막을 수 없는 이동은 초안 자동 보관(shared/lib/drafts)이 맡는다.
 */
const dirty = new Set<symbol>();

export function hasUnsavedEdits(): boolean {
  return dirty.size > 0;
}

function onBeforeUnload(event: BeforeUnloadEvent) {
  event.preventDefault();
  // 옛 브라우저는 returnValue 가 있어야 묻는다
  event.returnValue = "";
}

/** dirty 인 동안 이동 경고를 켠다. */
export function useLeaveGuard(isDirty: boolean) {
  useEffect(() => {
    if (!isDirty) { return undefined; }
    const token = Symbol("unsaved");
    dirty.add(token);
    if (dirty.size === 1) { window.addEventListener("beforeunload", onBeforeUnload); }
    return () => {
      dirty.delete(token);
      if (dirty.size === 0) { window.removeEventListener("beforeunload", onBeforeUnload); }
    };
  }, [isDirty]);
}
