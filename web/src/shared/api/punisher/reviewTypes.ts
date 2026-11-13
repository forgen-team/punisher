/** 간격 복습(P2) — punisher-server models/checks.go · learning/review.go 와 같은 형태. */
import type { CheckGrader, CheckSessionStatus } from "./checkTypes";

/** 복습 채점 방식(설정) — 자기 채점 / 튜터 채점 */
export type ReviewGrader = "self" | "llm";

/** GET /today 의 review(복습 큐 요약, 강의 추천과 별개). */
export interface TodayReview {
  /** 지금 만기된 복습 수(밀린 것 포함) */
  due: number;
  /** 오늘 이전에 만기된 수 */
  overdue: number;
  /** 유입 조절로 대기 중인 목표 수 */
  queued: number;
  /** 오늘 세션에 들어갈 만큼의 예상 분 */
  estimatedMinutes: number;
  /** 밀림이 처리량 3일치를 넘음("밀린 복습 n개" · '오늘 더 하기') */
  backlogged: boolean;
  /** 시간 합계의 강의 몫("복습 m분 + 강의 30분") */
  lectureMinutes: number;
  /** 오늘 세션에 섞을 '이미 앎' 확인 수(일괄 이미 앎 대기열) */
  verify: number;
}

/** 오늘의 최근 복습 세션(GET /reviews/summary 의 today). */
export interface ReviewSessionRef {
  id: string;
  status: CheckSessionStatus;
  grader: CheckGrader;
  total: number;
  done: number;
}

/** GET /reviews/summary */
export interface ReviewSummary {
  due: number;
  overdue: number;
  queued: number;
  estimatedMinutes: number;
  backlogged: boolean;
  /** 하루 처리량(문항 수) */
  throughput: number;
  grader: ReviewGrader;
  reviewMinutes: number;
  /** 오늘 세션에 섞을 '이미 앎' 확인 수 */
  verify: number;
  /** null = 오늘 세션 없음 */
  today: ReviewSessionRef | null;
}

/** GET /checks/history 의 세션 한 줄. */
export interface CheckHistoryItem {
  id: string;
  kind: "check" | "review";
  grader: CheckGrader;
  status: CheckSessionStatus;
  conceptId?: string;
  conceptName?: string;
  lectureId?: string;
  lectureTitle?: string;
  /** 복습 세션의 KST 날짜("2026-11-13") */
  reviewDay?: string;
  itemCount: number;
  doneCount: number;
  /** 혼자 맞힌(unaided_correct) 문항 수 */
  unaidedCorrect: number;
  createdAt: string;
  finishedAt?: string | null;
}

export interface CheckHistoryPage {
  sessions: CheckHistoryItem[];
  nextCursor: string;
}

/** 간격 × 채점 주체 한 칸(처음 보는 문항의 첫 시도). */
export interface RetentionBucket {
  intervalDays: number;
  grader: "llm" | "self";
  attempts: number;
  correct: number;
  /** 시도가 없으면 null */
  rate: number | null;
}

/** GET /mastery/metrics */
export interface MasteryMetrics {
  since: string;
  weeks: number;
  buckets: RetentionBucket[];
}

/** GET /mastery?conceptIds= 의 개념 한 줄(목표 목록 없음). */
export interface ConceptMasterySummary {
  conceptId: string;
  level: number;
  mastered: boolean;
  masteryTarget: number;
  uncheckedCount: number;
}

/** 일괄 숙달 조회 한 번의 개념 수 상한(서버 maxMasteryBatch). */
export const MASTERY_BATCH_MAX = 300;
/** 하루 복습 시간 범위(분, 서버 learning.ValidReviewMinutes). */
export const REVIEW_MINUTES_MAX = 30;
/** 복습 간격 단계 수 상한(서버 learning.MaxReviewInterval). */
export const REVIEW_INTERVALS_MAX = 6;
