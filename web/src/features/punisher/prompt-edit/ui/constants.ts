import type { PromptStage } from "@/src/shared/api/punisher";

/** 단계 순서(파이프라인 순) — 목록은 이 순서로 묶는다. resolve 는 '고급'으로 접는다. */
export const STAGE_ORDER: PromptStage[] = ["outline", "lecture", "review", "gloss", "qa", "explain", "resolve"];

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
