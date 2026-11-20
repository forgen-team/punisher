import {
  type CheckGrader,
  type CheckHistoryKind,
  type CheckOutcome,
  type CheckVerdict,
  type FeedbackKind,
  type MasteryLevel,
  type GoalStatus,
  type PunisherArea,
  type QuestionStatus,
  type RecommendationKind,
  type StudyRequestStatus,
  type UserConceptState,
  type ReviewGrader,
  type ChosenReaderLevel,
} from "@/src/shared/api/punisher";
import { getLocale, translate, type Locale } from "@/src/shared/i18n";
import { liveLabels } from "./liveLabels";
import { messages } from "../i18n";

export const AREA_LABELS_BY_LOCALE: Record<Locale, Record<PunisherArea, string>> = {
  ko: {
    foundations: "0단계 기초",
    "data-structures": "자료구조",
    "computer-architecture": "컴퓨터구조",
    os: "운영체제",
    "network-web": "네트워크/웹",
    database: "데이터베이스",
    distributed: "분산 시스템",
    "infra-ops": "인프라/운영",
    "security-design": "보안/설계",
  },
  en: {
    foundations: "Foundations",
    "data-structures": "Data structures",
    "computer-architecture": "Computer architecture",
    os: "Operating systems",
    "network-web": "Networking/Web",
    database: "Databases",
    distributed: "Distributed systems",
    "infra-ops": "Infra/Ops",
    "security-design": "Security/Design",
  },
};
export const AREA_LABELS = liveLabels(AREA_LABELS_BY_LOCALE);

export function areaLabel(area: string, locale: Locale = getLocale()): string {
  if (area === "other") { return translate(messages, locale).areaOther; }
  return AREA_LABELS_BY_LOCALE[locale][area as PunisherArea] ?? area;
}

export const USER_STATE_LABELS_BY_LOCALE: Record<Locale, Record<UserConceptState, string>> = {
  ko: { "not_started": "미학습", "in_progress": "진행 중", completed: "완료", known: "이미 앎" },
  en: { "not_started": "Not started", "in_progress": "In progress", completed: "Completed", known: "Already known" },
};
export const USER_STATE_LABELS = liveLabels(USER_STATE_LABELS_BY_LOCALE);

/** Map node display state (user state + graph flags). */
export type MapNodeState = "completed" | "known" | "in_progress" | "recommended" | "not_started" | "locked";

export const MAP_STATE_LABELS_BY_LOCALE: Record<Locale, Record<MapNodeState, string>> = {
  ko: {
    completed: "완료",
    known: "이미 앎",
    "in_progress": "진행 중",
    recommended: "추천(다음 학습)",
    "not_started": "미학습",
    locked: "잠김(선행 미완)",
  },
  en: {
    completed: "Completed",
    known: "Already known",
    "in_progress": "In progress",
    recommended: "Recommended (up next)",
    "not_started": "Not started",
    locked: "Locked (prerequisites pending)",
  },
};
export const MAP_STATE_LABELS = liveLabels(MAP_STATE_LABELS_BY_LOCALE);

/** 지도 범례·노드 표시 순서 (색 외에 아이콘 + 테두리 모양 + 텍스트로 구분: ui/MapStateIcon). */
export const MAP_STATE_ORDER: MapNodeState[] = ["completed", "known", "in_progress", "recommended", "not_started", "locked"];

export const RECOMMENDATION_KIND_LABELS_BY_LOCALE: Record<Locale, Record<RecommendationKind, string>> = {
  ko: {
    resume: "이어보기",
    "goal_path": "목표 경로",
    "study_request": "학습 요청",
    "essential_gap": "기본기 보강",
    "new_goal": "새 심화 목표",
  },
  en: {
    resume: "Continue",
    "goal_path": "Goal path",
    "study_request": "Study request",
    "essential_gap": "Fill a basics gap",
    "new_goal": "New deep-dive goal",
  },
};
export const RECOMMENDATION_KIND_LABELS = liveLabels(RECOMMENDATION_KIND_LABELS_BY_LOCALE);

/**
 * 추천 이유 문구. 단독판은 직접 입력 학습 요청을 유지하므로(docs/SYNC.md) 학습 요청 추천도
 * 서버 이유("학습 요청 '…'")를 그대로 보인다 — witems 는 중립 문구로 바꾼다.
 */
export function recommendationReason(item: { kind: RecommendationKind; reason?: string }): string {
  return item.reason ?? "";
}

export const STUDY_REQUEST_STATUS_LABELS_BY_LOCALE: Record<Locale, Record<StudyRequestStatus, string>> = {
  ko: {
    unresolved: "개념 찾는 중",
    resolved: "학습 가능",
    "in_progress": "학습 중",
    done: "완료",
    dismissed: "보류",
  },
  en: {
    unresolved: "Finding concepts",
    resolved: "Ready to learn",
    "in_progress": "Learning",
    done: "Done",
    dismissed: "On hold",
  },
};
export const STUDY_REQUEST_STATUS_LABELS = liveLabels(STUDY_REQUEST_STATUS_LABELS_BY_LOCALE);

export const STUDY_REQUEST_SOURCE_LABELS_BY_LOCALE: Record<Locale, Record<string, string>> = {
  ko: { rss: "RSS", manual: "직접 입력", guardian: "Guardian" },
  en: { rss: "RSS", manual: "Typed in", guardian: "Guardian" },
};
export const STUDY_REQUEST_SOURCE_LABELS = liveLabels(STUDY_REQUEST_SOURCE_LABELS_BY_LOCALE);

export const FEEDBACK_LABELS_BY_LOCALE: Record<Locale, Record<FeedbackKind, string>> = {
  ko: {
    understood: "이해 잘 됨",
    insufficient: "설명 부족",
    "too_easy": "너무 쉬움",
    "too_hard": "너무 어려움",
    "want_more": "더 알고 싶음",
  },
  en: {
    understood: "Clear",
    insufficient: "Needs more explanation",
    "too_easy": "Too easy",
    "too_hard": "Too hard",
    "want_more": "Want to know more",
  },
};
export const FEEDBACK_LABELS = liveLabels(FEEDBACK_LABELS_BY_LOCALE);

export const QUESTION_STATUS_LABELS_BY_LOCALE: Record<Locale, Record<QuestionStatus, string>> = {
  ko: { pending: "답변 생성 중", answered: "답변 완료", failed: "답변 실패" },
  en: { pending: "Writing answer", answered: "Answered", failed: "Answer failed" },
};
export const QUESTION_STATUS_LABELS = liveLabels(QUESTION_STATUS_LABELS_BY_LOCALE);

export const GOAL_STATUS_LABELS_BY_LOCALE: Record<Locale, Record<GoalStatus, string>> = {
  ko: { active: "진행 중", achieved: "달성", archived: "보관" },
  en: { active: "Active", achieved: "Achieved", archived: "Archived" },
};
export const GOAL_STATUS_LABELS = liveLabels(GOAL_STATUS_LABELS_BY_LOCALE);

export const SECTION_KIND_LABELS_BY_LOCALE: Record<Locale, Record<string, string>> = {
  ko: {
    "situation_question": "상황과 오늘의 질문",
    principles: "배경·원리",
    walkthrough: "코드·로그·도식 해설",
    "judgment_next": "판단 기준·다음 연결",
    // '이 질문에 답하기' 해설
    summary: "먼저 답",
    causes: "원인 사슬",
    assumptions: "내 환경 추정",
    verify: "확인 방법",
    fix: "대처",
    followup: "이어서 묻기",
  },
  en: {
    "situation_question": "Situation and today's question",
    principles: "Background and principles",
    walkthrough: "Code, logs, and diagrams explained",
    "judgment_next": "How to judge and what's next",
    summary: "Short answer",
    causes: "Chain of causes",
    assumptions: "Assumptions about your setup",
    verify: "How to verify",
    fix: "What to do",
    followup: "Ask a follow-up",
  },
};
export const SECTION_KIND_LABELS = liveLabels(SECTION_KIND_LABELS_BY_LOCALE);

export const INQUIRY_STATUS_LABELS_BY_LOCALE: Record<Locale, Record<string, string>> = {
  ko: { pending: "해설 쓰는 중", answered: "해설 완료", failed: "만들지 못함" },
  en: { pending: "Writing explanation", answered: "Explained", failed: "Couldn't create" },
};
export const INQUIRY_STATUS_LABELS = liveLabels(INQUIRY_STATUS_LABELS_BY_LOCALE);

export const RESOLUTION_METHOD_LABELS_BY_LOCALE: Record<Locale, Record<string, string>> = {
  ko: { alias: "이름/별칭 일치", llm: "AI 해석", proposed: "새 개념 제안", none: "해석 실패" },
  en: { alias: "Name/alias match", llm: "AI interpretation", proposed: "New concept proposed", none: "Couldn't interpret" },
};
export const RESOLUTION_METHOD_LABELS = liveLabels(RESOLUTION_METHOD_LABELS_BY_LOCALE);

/** 숙달 화면 레벨(design §2.2) */
export const MASTERY_LABELS_BY_LOCALE: Record<Locale, Record<MasteryLevel, string>> = {
  ko: { 0: "미확인", 1: "들어봄", 2: "따라감", 3: "혼자 답함", 4: "기억함", 5: "설명 가능" },
  en: { 0: "Unchecked", 1: "Heard of it", 2: "Can follow", 3: "Answers alone", 4: "Remembers", 5: "Can explain" },
};
export const MASTERY_LABELS = liveLabels(MASTERY_LABELS_BY_LOCALE);

/** 레벨 점의 전체 개수(0 = 점 없음, 5 = 다섯 개 모두 채움). */
export const MASTERY_MAX_LEVEL = 5;

export function masteryLabel(level: number, locale: Locale = getLocale()): string {
  return MASTERY_LABELS_BY_LOCALE[locale][level as MasteryLevel] ?? String(level);
}

/** 답 하나의 판정 */
export const VERDICT_LABELS_BY_LOCALE: Record<Locale, Record<CheckVerdict, string>> = {
  ko: { correct: "정답", partial: "부분 정답", incorrect: "오답", revealed: "답 공개", held: "판정 보류" },
  en: { correct: "Correct", partial: "Partly correct", incorrect: "Incorrect", revealed: "Answer revealed", held: "Verdict on hold" },
};
export const VERDICT_LABELS = liveLabels(VERDICT_LABELS_BY_LOCALE);

/** 문항 결과(문항이 끝날 때 하나로 정해짐) */
export const OUTCOME_LABELS_BY_LOCALE: Record<Locale, Record<CheckOutcome, string>> = {
  ko: {
    "unaided_correct": "혼자 맞힘",
    "unaided_partial": "일부 맞힘",
    aided: "도움받아 답함",
    failed: "놓침",
    held: "판정 보류",
    recorded: "기록함",
  },
  en: {
    "unaided_correct": "Got it alone",
    "unaided_partial": "Got part of it",
    aided: "Answered with help",
    failed: "Missed",
    held: "Verdict on hold",
    recorded: "Recorded",
  },
};
export const OUTCOME_LABELS = liveLabels(OUTCOME_LABELS_BY_LOCALE);

/** 마지막 근거의 채점 주체(레벨 점 옆) */
export const GRADER_LABELS_BY_LOCALE: Record<Locale, Record<CheckGrader, string>> = {
  ko: { llm: "튜터", self: "자기 채점", "llm_regrade": "재채점", none: "" },
  en: { llm: "Tutor", self: "Self-graded", "llm_regrade": "Regraded", none: "" },
};
export const GRADER_LABELS = liveLabels(GRADER_LABELS_BY_LOCALE);

/** 자기 채점에서 고르는 판정 */
export type SelfVerdict = "correct" | "partial" | "incorrect";

export const SELF_VERDICT_LABELS_BY_LOCALE: Record<Locale, Record<SelfVerdict, string>> = {
  ko: { correct: "맞음", partial: "부분", incorrect: "틀림" },
  en: { correct: "Correct", partial: "Partial", incorrect: "Incorrect" },
};
export const SELF_VERDICT_LABELS = liveLabels(SELF_VERDICT_LABELS_BY_LOCALE);

export const SELF_VERDICT_ORDER: SelfVerdict[] = ["correct", "partial", "incorrect"];

/** 세션 종류 */
export const CHECK_KIND_LABELS_BY_LOCALE: Record<Locale, Record<CheckHistoryKind, string>> = {
  ko: { check: "확인", review: "복습", "known_verify": "이미 앎 확인", synthesis: "종합 문제" },
  en: { check: "Check", review: "Review", "known_verify": "Known check", synthesis: "Synthesis" },
};
export const CHECK_KIND_LABELS = liveLabels(CHECK_KIND_LABELS_BY_LOCALE);

/** 설정의 복습 채점 방식 */
export const REVIEW_GRADER_LABELS_BY_LOCALE: Record<Locale, Record<ReviewGrader, string>> = {
  ko: { self: "자기 채점", llm: "튜터 채점" },
  en: { self: "Self-grading", llm: "Tutor grading" },
};
export const REVIEW_GRADER_LABELS = liveLabels(REVIEW_GRADER_LABELS_BY_LOCALE);

/** 설명 수준 이름(처음 설정 설문·설정 › 학습 선택지가 같은 문구). */
export const READER_LEVEL_LABELS_BY_LOCALE: Record<Locale, Record<ChosenReaderLevel, string>> = {
  ko: { beginner: "개발은 처음이에요", novice: "코드를 조금 써 봤어요", practitioner: "웹 개발을 실무로 해요", advanced: "CS 기본기는 탄탄해요" },
  en: { beginner: "I'm new to development", novice: "I've written a little code", practitioner: "I build for the web at work", advanced: "My CS fundamentals are solid" },
};
export const READER_LEVEL_LABELS = liveLabels(READER_LEVEL_LABELS_BY_LOCALE);

export { CHECK_ERROR_MESSAGES, SETTINGS_ERROR_MESSAGES, type CheckErrorCode, type SettingsErrorCode } from "./errorMessages";
