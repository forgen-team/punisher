
import { useRouter } from "@/src/shared/lib/router";
import { useStartCheck } from "@/src/entities/punisher";

export function checkHref(id: string): string {
  return `/check/${encodeURIComponent(id)}`;
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
