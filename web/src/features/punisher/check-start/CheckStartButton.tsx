
import { Button } from "@/src/shared/ui/kit";
import { describeCheckError } from "@/src/entities/punisher";
import { useLaunchCheck } from "./useLaunchCheck";

/** 강의 끝 '완료' 옆 '확인하기'. 메뉴 항목은 useLaunchCheck 를 같이 쓴다. */
export function CheckStartButton({ check }: { check: ReturnType<typeof useLaunchCheck> }) {
  return (
    <Button variant="secondary" onClick={check.launch} loading={check.isPending}>
      확인하기
    </Button>
  );
}

/** 시작 실패 한 줄(409 는 서버 안내 그대로). */
export function CheckStartError({ check }: { check: ReturnType<typeof useLaunchCheck> }) {
  if (!check.error) { return null; }
  return (
    <p role="alert" className="text-sm text-danger">
      {describeCheckError(check.error)}
    </p>
  );
}
