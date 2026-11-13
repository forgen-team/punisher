/** 학습 UX 확장 타입 — 학습 설정 · 선택 구절 설명 · 지도 위 학습 요청 (punisher-server 003). */
import type { ReviewGrader } from "./reviewTypes";

export type ExampleLanguage = "typescript" | "javascript" | "go" | "java" | "kotlin" | "python";

export interface LearningSettings {
  exampleLanguage: ExampleLanguage;
  /** 하루 학습량(분) — 15/30/45/60 */
  dailyMinutes: number;
  /** 숙달로 보는 레벨(3~5) */
  masteryTarget: number;
  /** 복습 간격(일, 증가하는 1~6개) */
  reviewIntervals: number[];
  /** 하루 복습 시간(0~30분) */
  reviewMinutes: number;
  reviewGrader: ReviewGrader;
  updatedAt?: string;
}

/** PUT /me/settings — 준 필드만 바뀐다. */
export type LearningSettingsPatch = Partial<Omit<LearningSettings, "updatedAt">>;

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

/** "이 개념 더 깊게"로 이어진 강의(GET /lectures/{id}/links). */
export interface LinkedLecture {
  id: string;
  conceptId: string;
  conceptName: string;
  title: string;
  kind: "library" | "instant";
  status: string;
  createdAt: string;
  linkedAt: string;
  /** 원 강의에서 "더 깊게"를 누른 위치 */
  anchor: string;
  percent: number;
  completedAt?: string;
  /** deeper: "이 개념 더 깊게" · regenerated: "새 프롬프트로 다시 만들기" */
  linkKind: "deeper" | "regenerated";
  promptVersion: string;
}

export interface LectureLinks {
  parents: LinkedLecture[];
  children: LinkedLecture[];
}
