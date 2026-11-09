
import { useState } from "react";
import { useDismissLecture, useRestoreLecture } from "@/src/entities/punisher";

export interface DismissedLecture {
  lectureId: string;
  label: string;
}

/**
 * 이어보기에서 제거 + 바로 되돌리기. 강의·진도는 지우지 않는다(서버 소프트 처리).
 * onChanged: 오늘 홈처럼 별도 쿼리 키를 쓰는 화면이 다시 불러오도록.
 */
export function useResumeDismiss(onChanged?: () => void) {
  const dismissMutation = useDismissLecture();
  const restoreMutation = useRestoreLecture();
  const [dismissed, setDismissed] = useState<DismissedLecture | null>(null);

  function dismiss(item: DismissedLecture) {
    dismissMutation.mutate(item.lectureId, {
      onSuccess: () => {
        setDismissed(item);
        onChanged?.();
      },
    });
  }

  function undo() {
    if (!dismissed) { return; }
    restoreMutation.mutate(dismissed.lectureId, {
      onSuccess: () => {
        setDismissed(null);
        onChanged?.();
      },
    });
  }

  return {
    dismiss,
    undo,
    dismissed,
    clear: () => setDismissed(null),
    pending: dismissMutation.isPending || restoreMutation.isPending,
    error: dismissMutation.error ?? restoreMutation.error,
  };
}
