
import { useEffect } from "react";

/**
 * 모달/오버레이가 열릴 때 body 스크롤을 잠그는 훅.
 * iOS Safari의 bounce scroll 문제까지 처리.
 */
export function useBodyScrollLock(isLocked: boolean) {
  useEffect(() => {
    if (!isLocked) return;

    const scrollY = window.scrollY;
    const originalStyle = {
      overflow: document.body.style.overflow,
      position: document.body.style.position,
      top: document.body.style.top,
      left: document.body.style.left,
      right: document.body.style.right,
      width: document.body.style.width,
    };

    // iOS Safari에서는 overflow: hidden만으로 부족하므로 position: fixed 사용
    document.body.style.overflow = "hidden";
    document.body.style.position = "fixed";
    document.body.style.top = `-${scrollY}px`;
    document.body.style.left = "0";
    document.body.style.right = "0";
    document.body.style.width = "100%";

    return () => {
      document.body.style.overflow = originalStyle.overflow;
      document.body.style.position = originalStyle.position;
      document.body.style.top = originalStyle.top;
      document.body.style.left = originalStyle.left;
      document.body.style.right = originalStyle.right;
      document.body.style.width = originalStyle.width;
      // 스크롤 위치 복원 — html 의 scroll-behavior: smooth 를 타면 맨 위에서 원래 위치까지
      // 미끄러지듯 다시 스크롤돼 어지럽다. 즉시 복원한다.
      window.scrollTo({ top: scrollY, left: 0, behavior: "instant" });
    };
  }, [isLocked]);
}
