import type { RetentionBucket } from "@/src/shared/api/punisher";

/** 기억 유지 한 숫자(%) — 모든 간격·채점 주체의 첫 시도 정답률. 시도가 없으면 null. */
export function overallRetention(buckets: RetentionBucket[]): number | null {
  const attempts = buckets.reduce((n, b) => n + b.attempts, 0);
  if (attempts === 0) { return null; }
  const correct = buckets.reduce((n, b) => n + b.correct, 0);
  return Math.round((correct / attempts) * 100);
}

/** "3/10" — 오늘 한 것/(한 것 + 남은 것). */
export function doneOfTotal(done: number, due: number): string {
  return `${done}/${done + due}`;
}
