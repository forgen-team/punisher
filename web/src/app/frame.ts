import type { PageWidth } from "@forgen-team/ui";
import { SECTION_HREFS, isSectionRoot, segmentOf } from "@/src/widgets/punisher/nav";

// 앱 틀(Page) 규칙 — 경로만 보고 정하는 순수 함수들(policy/20 §5·§6·§7, 재설계 punisher.md §1·§2).
// 두 셸(AppShell·WitemsShell)이 같은 표를 쓴다.

/** 본문 폭(policy/20 §7): 홈·캔버스·강의 = full, 질문 해설 = reading, 설정 = narrow, 프롬프트(2단) = wide. */
export function pageWidthFor(pathname: string): PageWidth {
  const path = pathname || "/";
  if (path === "/" || path === "/map" || path.startsWith("/map/") || path.startsWith("/lectures/")) { return "full"; }
  if (path.startsWith("/inquiries/")) { return "reading"; }
  if (path === "/settings") { return "narrow"; }
  if (path === "/settings/prompts") { return "wide"; }
  return "default";
}

/**
 * ← 가 주소로 바로 들어왔을 때 갈 곳(policy/20 §5 목적지 2) — 그 경로가 속한 섹션 첫 화면, 섹션 밖이면 처음("/").
 * 섹션 첫 화면이면 null(← 없음).
 */
export function backFallbackFor(pathname: string): string | null {
  if (isSectionRoot(pathname)) { return null; }
  const segment = segmentOf(pathname);
  return segment ? SECTION_HREFS[segment] : "/";
}

/** Claude 상태 배너를 둘 화면 — 생성을 쓰는 곳만(오늘·강의·복습 세션·질문 해설·새 챕터·생성 중·강의 시작). */
export function usesGeneration(pathname: string): boolean {
  const path = pathname || "/";
  return (
    path === "/"
    || path === "/chapters/new"
    || path === "/start"
    || ["/lectures/", "/check/", "/inquiries/", "/instant/"].some((prefix) => path.startsWith(prefix))
  );
}
