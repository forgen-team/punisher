import { useEffect } from "react";
import { useAtomValue } from "jotai";
import { themePreferenceAtom, type ThemePreference } from "@/src/shared/lib/theme";

function applyTheme(preference: ThemePreference): void {
  const dark =
    preference === "dark" || (preference === "system" && window.matchMedia("(prefers-color-scheme: dark)").matches);
  document.documentElement.setAttribute("data-theme", dark ? "dark" : "light");
}

/**
 * `<html data-theme>`를 항상 실제 테마(light|dark)로 유지한다. 지식 지도(React Flow)·코드 하이라이트·mermaid 는
 * JS 로 data-theme 을 읽으므로 속성이 비면 라이트로 그려진다. 설정 화면 변경·시스템 설정 변경 모두 반영한다.
 * (첫 페인트 전 적용은 index.html 의 인라인 스크립트가 맡는다.)
 */
export function ThemeSync(): null {
  const preference = useAtomValue(themePreferenceAtom);
  useEffect(() => {
    applyTheme(preference);
    const media = window.matchMedia("(prefers-color-scheme: dark)");
    const onChange = () => applyTheme(preference);
    media.addEventListener("change", onChange);
    return () => media.removeEventListener("change", onChange);
  }, [preference]);
  return null;
}
