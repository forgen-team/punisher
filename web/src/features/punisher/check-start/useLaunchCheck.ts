
import { useRouter } from "@/src/shared/lib/router";
import { useStartCheck } from "@/src/entities/punisher";
import { PunisherApiError } from "@/src/shared/api/punisher";

export function checkHref(id: string): string {
  return `/check/${encodeURIComponent(id)}`;
}

/** 검수 통과 기록이 없는 강의의 '확인하기'(409 LECTURE_NOT_REVIEWED) — 오류가 아니라 상태 한 줄로 보인다. */
export function isLectureNotReviewed(error: unknown): boolean {
  return error instanceof PunisherApiError && error.status === 409 && error.code === "LECTURE_NOT_REVIEWED";
}

/** '확인하기' — 세션을 받고(열린 세션이 있으면 그것) 확인 화면으로 간다. */
export function useLaunchCheck(lectureId: string) {
  const router = useRouter();
  const start = useStartCheck();
  return {
    launch: () => start.mutate(lectureId, { onSuccess: (detail) => router.push(checkHref(detail.session.id)) }),
    isPending: start.isPending,
    error: start.error,
  };
}
