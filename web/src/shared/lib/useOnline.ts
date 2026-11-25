import { useSyncExternalStore } from "react";

function subscribe(listener: () => void) {
  window.addEventListener("online", listener);
  window.addEventListener("offline", listener);
  return () => {
    window.removeEventListener("online", listener);
    window.removeEventListener("offline", listener);
  };
}

/** 브라우저가 알려 주는 온라인 상태(오프라인 읽기 — 쓰기 동작을 막고 안내할 때). */
export function useOnline(): boolean {
  return useSyncExternalStore(subscribe, () => navigator.onLine, () => true);
}
