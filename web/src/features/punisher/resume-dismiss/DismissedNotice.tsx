
import { Button } from "@/src/shared/ui/kit";
import { describePunisherError } from "@/src/shared/api/punisher";
import { useT } from "@/src/shared/i18n";
import { messages } from "./i18n";
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
  const t = useT(messages);
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
      <span className="min-w-0 flex-1 truncate">{t.dismissed(dismissed.label)}</span>
      <Button variant="ghost" size="sm" loading={pending} onClick={onUndo}>
        {t.undo}
      </Button>
      <Button variant="ghost" size="sm" onClick={onClose}>
        {t.close}
      </Button>
    </div>
  );
}
