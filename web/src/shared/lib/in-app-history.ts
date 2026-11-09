
import { useEffect, useRef } from "react";

/**
 * 앱 안 이동 횟수 — 첫 화면 이후 주소가 바뀔 때마다 센다(RSS 마이크로앱의 pushState 도 Next 경로로 잡힌다).
 * 0 이면 이 탭에서 직접 들어온 것(새 탭·북마크·외부 링크) → 하위 화면의 ← 는 기본 상위 화면으로 간다.
 */
let inAppNavigations = 0;
/** 새로고침 뒤에도 같은 판단을 하도록 sessionStorage 에도 남긴다(witems 는 RSS 마이크로앱과 공유). */
const IN_APP_NAV_KEY = "punisher:in-app-nav";

/** 앱 안에서 들어왔는가(브라우저 뒤로가기로 들어온 곳에 돌아갈 수 있는가). */
export function canGoBack(): boolean {
  return inAppNavigations > 0;
}

/** 셸이 경로 변화를 세도록 한 번 호출한다(AppShell). */
export function useTrackInAppNavigation(pathname: string | null) {
  const first = useRef(true);
  useEffect(() => {
    if (first.current) {
      first.current = false;
      return;
    }
    inAppNavigations += 1;
    try {
      window.sessionStorage.setItem(IN_APP_NAV_KEY, String(inAppNavigations));
    } catch {
      // 저장소를 못 쓰면 마이크로앱은 자기 라우터 기록만으로 판단한다
    }
  }, [pathname]);
}
