/**
 * 끝까지 읽었지만 '완료'를 누르지 않은 강의 — 서버는 계속 진행 중(completed_at 없음)으로 두고,
 * 화면에서만 "100%"·"남은 시간" 대신 '다 읽음'으로 보인다.
 */
export function isReadThrough(percent: number | undefined, completed: boolean): boolean {
  return !completed && (percent ?? 0) >= 100;
}

export const READ_THROUGH_LABEL = "다 읽음";
