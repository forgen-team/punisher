/** 학습 UX 확장 타입 — 학습 설정 · 선택 구절 설명 · 지도 위 학습 요청 (punisher-server 003). */

export type ExampleLanguage = "typescript" | "javascript" | "go" | "java" | "kotlin" | "python";

export interface LearningSettings {
  exampleLanguage: ExampleLanguage;
  updatedAt?: string;
}

export interface ExampleLanguageOption {
  id: ExampleLanguage;
  label: string;
}

export interface LearningSettingsResponse {
  settings: LearningSettings;
  exampleLanguages: ExampleLanguageOption[];
}

/** POST /lectures/{id}/explain — 저장되지 않는 해석(LLM). */
export interface SelectionExplanation {
  selection: string;
  sectionTitle: string;
  explanation: string;
  model: string;
}

/** 선택 구절 길이 제한(서버 services.MaxSelectionRunes 와 같게). */
export const SELECTION_MIN_CHARS = 2;
export const SELECTION_MAX_CHARS = 1000;

/** 지식 지도 위의 대기 학습 요청. */
export interface MapRequest {
  id: string;
  query: string;
  source: "rss" | "manual" | "guardian";
  status: "unresolved" | "resolved" | "in_progress";
}
