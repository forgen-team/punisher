import type { PromptStage } from "@/src/shared/api/punisher";

/** 단계 순서(파이프라인 순) — 목록은 이 순서로 묶는다. resolve 는 '고급'으로 접는다. */
export const STAGE_ORDER: PromptStage[] = ["outline", "lecture", "review", "gloss", "qa", "explain", "check", "pretest", "resolve"];

export const HISTORY_ACTION_LABELS: Record<"save" | "reset" | "restore", string> = {
  save: "저장",
  reset: "기본값으로 되돌리기 전",
  restore: "복원",
};

export type EditorTab = "edit" | "diff" | "preview";

export const EDITOR_TABS: { value: EditorTab; label: string }[] = [
  { value: "edit", label: "편집" },
  { value: "diff", label: "기본값과 비교" },
  { value: "preview", label: "미리보기" },
];

/**
 * 편집기 라벨(헤더·패널에는 제목만 두고 적용 범위는 제목 옆 정보 버튼의 팝오버로).
 * 단독판: 라이브러리(추천 강의 미리 만들기)도 사용자 프롬프트를 쓴다(LibraryUsesUserPrompts, docs/SYNC.md).
 */
export const EDITOR_LABELS = {
  body: "내 본문",
  appliesToTitle: "적용 범위",
  appliesTo: (scope: string) => `적용: ${scope} · 추천 강의 미리 만들기에도 적용, 이미 만든 강의는 그대로`,
} as const;
