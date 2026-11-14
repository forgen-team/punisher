
import { Button } from "@/src/shared/ui/kit";
import { describeCheckError } from "@/src/entities/punisher";
import { isLectureNotReviewed, useLaunchCheck } from "./useLaunchCheck";

/**
 * 강의 끝 '완료' 옆 '확인하기'. 메뉴 항목은 useLaunchCheck 를 같이 쓴다.
 * primary: 완료한 뒤에는 이 버튼이 주 버튼 — 확인하기를 풀어야 복습 일정이 잡힌다.
 */
export function CheckStartButton({ check, primary = false }: { check: ReturnType<typeof useLaunchCheck>; primary?: boolean }) {
  return (
    <Button variant={primary ? "primary" : "secondary"} onClick={check.launch} loading={check.isPending}>
      확인하기
    </Button>
  );
}

/** 시작 실패 한 줄(409 는 서버 안내 그대로, 검수 전 강의는 상태 한 줄). */
export function CheckStartError({ check }: { check: ReturnType<typeof useLaunchCheck> }) {
  if (!check.error) { return null; }
  if (isLectureNotReviewed(check.error)) {
    return <p role="status" className="text-sm text-fg-secondary">검수 후 확인할 수 있어요</p>;
  }
  return (
    <p role="alert" className="text-sm text-danger">
      {describeCheckError(check.error)}
    </p>
  );
}
