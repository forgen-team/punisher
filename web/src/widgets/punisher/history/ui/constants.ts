import type { CheckSessionStatus } from "@/src/shared/api/punisher";

/** 기록의 세션 상태(마친 세션은 표시 없음) */
export const HISTORY_STATUS_LABELS: Record<CheckSessionStatus, string> = {
  preparing: "준비 중",
  active: "진행 중",
  finished: "",
  abandoned: "그만둠",
  failed: "준비 실패",
};

/** 기억 유지 지표의 채점 주체 열 */
export const METRIC_GRADER_LABELS: Record<"llm" | "self", string> = {
  llm: "튜터 채점",
  self: "자기 채점",
};

export const METRIC_GRADER_ORDER: ("llm" | "self")[] = ["self", "llm"];
