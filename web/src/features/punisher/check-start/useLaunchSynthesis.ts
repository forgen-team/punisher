import { useRouter } from "@/src/shared/lib/router";
import { useStartDiagnostic } from "@/src/entities/punisher";
import { checkHref } from "./useLaunchCheck";

/** '이번 주 종합 문제' — 세션을 받거나(진행 중이면 이어서) 확인 화면으로 간다. 문항은 그때 만든다. */
export function useLaunchSynthesis() {
  const router = useRouter();
  const start = useStartDiagnostic();
  return {
    launch: () =>
      start.mutate({ kind: "synthesis" }, {
        onSuccess: (detail) => {
          if (detail) { router.push(checkHref(detail.session.id)); }
        },
      }),
    isPending: start.isPending,
    error: start.error,
  };
}
