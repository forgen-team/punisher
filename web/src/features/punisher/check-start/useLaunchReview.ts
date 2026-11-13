
import { useRouter } from "@/src/shared/lib/router";
import { useStartReview } from "@/src/entities/punisher";
import { checkHref } from "./useLaunchCheck";

/** '복습 시작'·'오늘 더 하기' — 오늘 복습 세션을 받고 확인 화면(check/[id])으로 간다. */
export function useLaunchReview() {
  const router = useRouter();
  const start = useStartReview();
  return {
    launch: () => start.mutate(undefined, { onSuccess: (detail) => router.push(checkHref(detail.session.id)) }),
    isPending: start.isPending,
    error: start.error,
  };
}
