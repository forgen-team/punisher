import { useToast } from "@forgen-team/ui";
import { useDismissLecture, useRestoreLecture } from "@/src/entities/punisher";
import { describePunisherError } from "@/src/shared/api/punisher";
import { useT } from "@/src/shared/i18n";
import { messages } from "./i18n";

/**
 * 이어보기에서 빼기 — 바로 빼고 토스트 "이어보기에서 뺐어요 [되돌리기]"(정책 23 §3: 되돌리기는 토스트 하나로).
 * 강의·진도는 지우지 않는다(서버 소프트 처리). 오늘·내 강의·지도 상태는 엔티티 훅이 다시 읽는다.
 */
export function useResumeDismiss() {
  const t = useT(messages);
  const toast = useToast();
  const dismissMutation = useDismissLecture();
  const restoreMutation = useRestoreLecture();

  function dismiss(lectureId: string) {
    dismissMutation.mutate(lectureId, {
      onSuccess: () => {
        toast.undo(t.dismissed, () => {
          restoreMutation.mutate(lectureId, { onError: () => toast.show(t.undoFailed, { tone: "danger" }) });
        }, t.undo);
      },
      onError: (error) => toast.show(describePunisherError(error), { tone: "danger" }),
    });
  }

  return { dismiss, pending: dismissMutation.isPending || restoreMutation.isPending };
}
