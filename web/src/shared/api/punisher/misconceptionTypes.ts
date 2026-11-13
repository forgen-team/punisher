/** 개인화(P4) — punisher-server models/misconceptions.go · learning/today.go(RecheckCard)와 같은 형태. */

export type MisconceptionStatus = "open" | "resolved";

/** 확인 결과에서 인정한 오개념(텍스트는 그 턴의 grade.misconception). */
export interface Misconception {
  id: string;
  conceptId: string;
  conceptName: string;
  /** "" = 통합 문항 */
  objectiveId: string;
  objectiveText: string;
  text: string;
  turnId: string;
  status: MisconceptionStatus;
  createdAt: string;
  resolvedAt?: string | null;
}

/** GET /today 의 recheck — 목표 경로 다음 개념의 선행 중 회복 안 된 반복 실패가 있는 개념("○○ 다시 확인"). */
export interface RecheckCard {
  conceptId: string;
  conceptName: string;
  area: string;
  goalId: string;
  /** 그 목표 중 하나가 지금 복습 만기 → 복습 세션으로 */
  due: boolean;
  /** 만기가 아니면 이 강의로 확인하기("" 또는 없음 = 강의 없음) */
  lectureId?: string;
}
