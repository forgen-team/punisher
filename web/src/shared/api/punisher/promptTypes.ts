/** 강의 프롬프트 지시문 블록(학습 설정 > 강의 프롬프트) — punisher-server prompts.Blocks / handlers/prompts.go. */

export type PromptStage = "outline" | "lecture" | "review" | "gloss" | "qa" | "explain" | "resolve" | "check" | "pretest";

export interface PromptVariable {
  name: string;
  description: string;
}

export interface PromptBlock {
  id: string;
  stage: PromptStage;
  stageTitle: string;
  title: string;
  /** 이 단계가 하는 일(한 줄) */
  purpose: string;
  /** 내 수정이 적용되는 곳 */
  appliesTo: string;
  warning?: string;
  advanced: boolean;
  vars: PromptVariable[];
  default: string;
  /** 지금 쓰이는 본문(수정했으면 내 본문) */
  body: string;
  overridden: boolean;
  updatedAt?: string;
}

/** 강의 피드백 종류 → 프롬프트에 넣을 제안 문구(StyleHintFor) */
export type FeedbackHints = Partial<Record<"too_easy" | "too_hard" | "insufficient" | "want_more", string>>;

export interface PromptsResponse {
  blocks: PromptBlock[];
  feedbackHints: FeedbackHints;
  version: string;
}

export interface PromptSaveResponse {
  block: PromptBlock;
  /** 이 블록이 모르는 {{변수}} — 그대로 남는다 */
  unknownVars?: string[];
}

export interface PromptHistoryEntry {
  id: number;
  promptId: string;
  body: string;
  action: "save" | "reset" | "restore";
  createdAt: string;
}

/** 미리보기 — 예시 개념으로 조립한 최종 프롬프트(LLM 호출 없음) */
export interface PromptPreview {
  blockId: string;
  stage: PromptStage;
  conceptId: string;
  conceptName: string;
  system: string;
  user: string;
  version: string;
  unknownVars: string[];
}
