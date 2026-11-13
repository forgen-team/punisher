import {
  CHECK_MAX_ANSWER_CHARS,
  CHECK_MAX_DISPUTE_CHARS,
  type CheckGrader,
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
} from "@/src/shared/api/punisher";

export const AREA_LABELS: Record<PunisherArea, string> = {
  "data-structures": "자료구조",
  "computer-architecture": "컴퓨터구조",
  os: "운영체제",
  "network-web": "네트워크/웹",
  database: "데이터베이스",
  distributed: "분산 시스템",
  "infra-ops": "인프라/운영",
  "security-design": "보안/설계",
};

export function areaLabel(area: string): string {
  if (area === "other") { return "기타"; }
  return AREA_LABELS[area as PunisherArea] ?? area;
}

export const USER_STATE_LABELS: Record<UserConceptState, string> = {
  "not_started": "미학습",
  "in_progress": "진행 중",
  completed: "완료",
  known: "이미 앎",
};

/** Map node display state (user state + graph flags). */
export type MapNodeState = "completed" | "known" | "in_progress" | "recommended" | "not_started" | "locked";

export const MAP_STATE_LABELS: Record<MapNodeState, string> = {
  completed: "완료",
  known: "알고 있음",
  "in_progress": "진행 중",
  recommended: "추천(다음 학습)",
  "not_started": "미학습",
  locked: "잠김(선행 미완)",
};

/** 지도 범례·노드 표시 순서 (색 외에 아이콘 + 테두리 모양 + 텍스트로 구분: ui/MapStateIcon). */
export const MAP_STATE_ORDER: MapNodeState[] = ["completed", "known", "in_progress", "recommended", "not_started", "locked"];

export const RECOMMENDATION_KIND_LABELS: Record<RecommendationKind, string> = {
  resume: "이어보기",
  "goal_path": "목표 경로",
  "study_request": "학습 요청",
  "essential_gap": "기본기 보강",
  "new_goal": "새 심화 목표",
};

export const STUDY_REQUEST_STATUS_LABELS: Record<StudyRequestStatus, string> = {
  unresolved: "개념 찾는 중",
  resolved: "학습 가능",
  "in_progress": "학습 중",
  done: "완료",
  dismissed: "보류",
};

export const STUDY_REQUEST_SOURCE_LABELS: Record<string, string> = {
  rss: "RSS",
  manual: "직접 입력",
  guardian: "Guardian",
};

export const FEEDBACK_LABELS: Record<FeedbackKind, string> = {
  understood: "이해 잘 됨",
  insufficient: "설명 부족",
  "too_easy": "너무 쉬움",
  "too_hard": "너무 어려움",
  "want_more": "더 알고 싶음",
};

export const QUESTION_STATUS_LABELS: Record<QuestionStatus, string> = {
  pending: "답변 생성 중",
  answered: "답변 완료",
  failed: "답변 실패",
};

export const GOAL_STATUS_LABELS: Record<GoalStatus, string> = {
  active: "진행 중",
  achieved: "달성",
  archived: "보관",
};

export const SECTION_KIND_LABELS: Record<string, string> = {
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
};

export const INQUIRY_STATUS_LABELS: Record<string, string> = {
  pending: "해설 쓰는 중",
  answered: "해설 완료",
  failed: "만들지 못함",
};

export const RESOLUTION_METHOD_LABELS: Record<string, string> = {
  alias: "이름/별칭 일치",
  llm: "AI 해석",
  proposed: "새 개념 제안",
  none: "해석 실패",
};

/** 숙달 화면 레벨(design §2.2) */
export const MASTERY_LABELS: Record<MasteryLevel, string> = {
  0: "미확인",
  1: "들어봄",
  2: "따라감",
  3: "혼자 답함",
  4: "기억함",
  5: "설명 가능",
};

/** 레벨 점의 전체 개수(0 = 점 없음, 5 = 다섯 개 모두 채움). */
export const MASTERY_MAX_LEVEL = 5;

export function masteryLabel(level: number): string {
  return MASTERY_LABELS[level as MasteryLevel] ?? String(level);
}

/** 답 하나의 판정 */
export const VERDICT_LABELS: Record<CheckVerdict, string> = {
  correct: "정답",
  partial: "부분 정답",
  incorrect: "오답",
  revealed: "답 공개",
  held: "판정 보류",
};

/** 문항 결과(문항이 끝날 때 하나로 정해짐) */
export const OUTCOME_LABELS: Record<CheckOutcome, string> = {
  "unaided_correct": "혼자 맞힘",
  "unaided_partial": "일부 맞힘",
  aided: "도움받아 답함",
  failed: "놓침",
  held: "판정 보류",
};

/** 마지막 근거의 채점 주체(레벨 점 옆) */
export const GRADER_LABELS: Record<CheckGrader, string> = {
  llm: "튜터",
  self: "자기",
  "llm_regrade": "재채점",
};

/** 자기 채점에서 고르는 판정 */
export type SelfVerdict = "correct" | "partial" | "incorrect";

export const SELF_VERDICT_LABELS: Record<SelfVerdict, string> = {
  correct: "맞음",
  partial: "부분",
  incorrect: "틀림",
};

export const SELF_VERDICT_ORDER: SelfVerdict[] = ["correct", "partial", "incorrect"];

/** 세션 종류 */
export const CHECK_KIND_LABELS: Record<"check" | "review", string> = {
  check: "확인",
  review: "복습",
};

/** 설정의 복습 채점 방식 */
export const REVIEW_GRADER_LABELS: Record<ReviewGrader, string> = {
  self: "자기 채점",
  llm: "튜터 채점",
};

/** 확인·복습 API 오류 code → 안내(describeCheckError) */
export type CheckErrorCode =
  | "ANSWER_EMPTY"
  | "ANSWER_TOO_LONG"
  | "REASON_TOO_LONG"
  | "INVALID_VERDICT"
  | "NO_OBJECTIVES"
  | "INVALID_KIND"
  | "NO_REVIEWS"
  | "BAD_REQUEST";

export const CHECK_ERROR_MESSAGES: Record<CheckErrorCode, string> = {
  ANSWER_EMPTY: "답을 적어 주세요.",
  ANSWER_TOO_LONG: `답은 ${CHECK_MAX_ANSWER_CHARS}자까지 쓸 수 있어요.`,
  REASON_TOO_LONG: `사유는 ${CHECK_MAX_DISPUTE_CHARS}자까지 쓸 수 있어요.`,
  INVALID_VERDICT: "맞음·부분·틀림 중에서 골라 주세요.",
  NO_OBJECTIVES: "이 강의에는 확인할 학습 목표가 없어요.",
  INVALID_KIND: "지원하지 않는 확인 종류예요.",
  NO_REVIEWS: "오늘 복습할 항목이 없어요.",
  BAD_REQUEST: "요청을 처리하지 못했어요. 다시 시도해 주세요.",
};
