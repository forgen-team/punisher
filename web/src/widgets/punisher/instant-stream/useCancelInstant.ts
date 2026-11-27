import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useToast } from "@forgen-team/ui";
import { useRouter } from "@/src/shared/lib/router";
import { invalidateLearningState } from "@/src/entities/punisher";
import { PunisherApiError, punisherApi } from "@/src/shared/api/punisher";
import { useLocale, translate } from "@/src/shared/i18n";
import { messages } from "./i18n";

/**
 * 생성 중 [취소] — 확인 창 없이 바로 취소하고 오늘(`/`)로 replace + 토스트.
 * 되돌릴 수는 없지만 같은 개념으로 다시 시작할 수 있어 확인을 묻지 않는다(정책 23 §3). 실패하면 인라인 오류.
 */
export function useCancelInstant(jobId: string) {
  const t = translate(messages, useLocale());
  const router = useRouter();
  const toast = useToast();
  const queryClient = useQueryClient();
  const mutation = useMutation({
    mutationFn: () => punisherApi.cancelJob(jobId),
    onSuccess: () => {
      void invalidateLearningState(queryClient);
      toast.show(t.cancelledToast);
      router.replace("/");
    },
  });
  let error = "";
  if (mutation.error) {
    error = mutation.error instanceof PunisherApiError && mutation.error.status === 409 ? t.cancelTooLate : t.cancelFailed;
  }
  return { cancel: () => mutation.mutate(), isPending: mutation.isPending, error };
}
