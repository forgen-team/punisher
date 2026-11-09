
import { Button } from "@/src/shared/ui/kit";
import { describePunisherError } from "@/src/shared/api/punisher";
import type { DismissedLecture } from "./useResumeDismiss";

interface DismissedNoticeProps {
  dismissed: DismissedLecture | null;
  error: unknown;
  pending: boolean;
  onUndo: () => void;
  onClose: () => void;
}

/** 제거 직후 한 줄 안내 + 되돌리기 (토스트에는 동작 버튼이 없어 목록 안에 둔다). */
export function DismissedNotice({ dismissed, error, pending, onUndo, onClose }: DismissedNoticeProps) {
  if (error) {
    return (
      <p role="alert" className="py-2 text-xs text-danger">
        {describePunisherError(error)}
      </p>
    );
  }
  if (!dismissed) { return null; }
  return (
    <div role="status" className="flex items-center gap-2 py-1 text-sm text-fg-secondary">
      <span className="min-w-0 flex-1 truncate">‘{dismissed.label}’을(를) 이어보기에서 뺐어요. 다시 열면 돌아와요.</span>
      <Button variant="ghost" size="sm" loading={pending} onClick={onUndo}>
        되돌리기
      </Button>
      <Button variant="ghost" size="sm" onClick={onClose}>
        닫기
      </Button>
    </div>
  );
}
