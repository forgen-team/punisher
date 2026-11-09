import type {
  FeedbackKind,
  GoalStatus,
  PunisherArea,
  QuestionStatus,
  RecommendationKind,
  StudyRequestStatus,
  UserConceptState,
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
  unresolved: "개념 확인 중",
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
};

export const RESOLUTION_METHOD_LABELS: Record<string, string> = {
  alias: "이름/별칭 일치",
  llm: "AI 해석",
  proposed: "새 개념 제안",
  none: "해석 실패",
};
