import type { PromptStage } from "@/src/shared/api/punisher";

/** 단계 순서(파이프라인 순) — 목록은 이 순서로 묶는다. resolve 는 '고급'으로 접는다. */
export const STAGE_ORDER: PromptStage[] = ["outline", "lecture", "review", "gloss", "qa", "explain", "check", "pretest", "resolve"];

export type EditorTab = "edit" | "diff" | "preview";

export const EDITOR_TABS: EditorTab[] = ["edit", "diff", "preview"];
