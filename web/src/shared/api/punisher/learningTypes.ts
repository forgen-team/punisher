/** 학습 UX 확장 타입 — 학습 설정 · 선택 구절 설명 · 지도 위 학습 요청 (punisher-server 003). */
import type { ReviewGrader } from "./reviewTypes";
import type { PunisherArea } from "./types";
import type { Locale } from "@/src/shared/i18n";

/** 강의 생성·검수·질문 답·채점을 맡는 AI(⑥) — 표시 이름·상태는 shared/api/status. */
export type AiProviderId = "claude" | "codex" | "agy";

/** 설명 수준(학습자 프로필) — "" 은 아직 정하지 않음(처음 설정 시트를 띄운다. 생성은 practitioner 로 취급). */
export type ReaderLevel = "" | "beginner" | "novice" | "practitioner" | "advanced";
export type ChosenReaderLevel = Exclude<ReaderLevel, "">;
/** PUT 이 받는 4개 값(쉬운 순). "" 은 보낼 수 없다(400 INVALID_READER_LEVEL). */
export const READER_LEVELS: readonly ChosenReaderLevel[] = ["beginner", "novice", "practitioner", "advanced"];
/** 건너뛰기·미정일 때의 기본 수준(서버 생성 기본값과 같다). */
export const DEFAULT_READER_LEVEL: ChosenReaderLevel = "practitioner";

/** 영역별 아는 정도 — can_explain 영역은 깊이 1 을 이미 앎으로 둔다. */
export type Familiarity = "new" | "heard" | "can_explain";
export const FAMILIARITY_LEVELS: readonly Familiarity[] = ["new", "heard", "can_explain"];
export type AreaFamiliarity = Partial<Record<PunisherArea, Familiarity>>;

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
  /** 강의 전 예측 질문 */
  pretestEnabled: boolean;
  /** 실습 환경(v3 §4) */
  labOs: LabOS;
  /** 목표마다 복습 간격을 조절(v3 §7) */
  adaptiveIntervals: boolean;
  /** 화면·생성 언어 */
  locale: Locale;
  /** 설명 수준 — "" 은 아직 정하지 않음 */
  readerLevel: ReaderLevel;
  /** 영역별 아는 정도(빈 객체 허용) */
  areaFamiliarity: AreaFamiliarity;
  /** 고른 AI — 없으면 예전 서버(claude). 바꾸면 다음 호출부터 바로 쓴다 */
  aiProvider?: AiProviderId;
  updatedAt?: string;
}

/** GET·PUT /me/settings 응답 최상위의 수준 보정 제안 — 바꾸기는 설정 PUT, 닫기는 POST /me/level-suggestion/dismiss(7일). */
export interface LevelSuggestion {
  to: ChosenReaderLevel;
  reason: string;
}

export type LabOS = "macos" | "linux" | "none";

/** PUT /me/settings — 준 필드만 바뀐다. */
export type LearningSettingsPatch = Partial<Omit<LearningSettings, "updatedAt" | "readerLevel">> & {
  readerLevel?: ChosenReaderLevel;
};

export interface ExampleLanguageOption {
  id: ExampleLanguage;
  label: string;
}

export interface LearningSettingsResponse {
  settings: LearningSettings;
  exampleLanguages: ExampleLanguageOption[];
  /** 수준 보정 제안(④) — 응답 최상위(settings 의 형제). 없으면 null */
  levelSuggestion?: LevelSuggestion | null;
}

/** 수준 보정 제안(응답 최상위) — 모르는 수준이거나 이미 그 수준이면 무시한다. */
export function levelSuggestionOf(res: LearningSettingsResponse | undefined): LevelSuggestion | null {
  const suggestion = res?.levelSuggestion ?? null;
  if (!suggestion || !READER_LEVELS.includes(suggestion.to)) { return null; }
  return suggestion.to === res?.settings.readerLevel ? null : suggestion;
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
  /** 강의 판 번호(다시 만든 판을 가른다) */
  version: number;
}

export interface LectureLinks {
  parents: LinkedLecture[];
  children: LinkedLecture[];
}
