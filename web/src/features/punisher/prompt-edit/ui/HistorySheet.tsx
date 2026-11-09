
import { BottomSheet, Button, EmptyState, ErrorState, ListSkeleton } from "@/src/shared/ui/kit";
import { usePromptHistory } from "@/src/entities/punisher";
import { describePunisherError, type PromptHistoryEntry } from "@/src/shared/api/punisher";
import { HISTORY_ACTION_LABELS } from "./constants";

const dateFormat = new Intl.DateTimeFormat("ko-KR", { dateStyle: "medium", timeStyle: "short" });

interface HistorySheetProps {
  blockId: string;
  open: boolean;
  restoring: boolean;
  onClose: () => void;
  onRestore: (entry: PromptHistoryEntry) => void;
  onLoad: (entry: PromptHistoryEntry) => void;
}

/** 이전 버전(최근 20개) — 편집기로 불러오거나 바로 복원. */
export function HistorySheet({ blockId, open, restoring, onClose, onRestore, onLoad }: HistorySheetProps) {
  const history = usePromptHistory(open ? blockId : "");
  const items = history.data ?? [];
  return (
    <BottomSheet open={open} onClose={onClose} title="이전 버전" description="최근 20개까지 남아요" desktop="side">
      {history.isLoading ? (
        <ListSkeleton rows={3} />
      ) : history.error ? (
        <ErrorState compact message={describePunisherError(history.error)} onRetry={() => history.refetch()} />
      ) : items.length === 0 ? (
        <EmptyState compact title="아직 저장한 버전이 없어요" />
      ) : (
        <ul role="list" className="flex flex-col gap-3">
          {items.map((entry) => (
            <li key={entry.id} className="rounded-md border border-line p-3">
              <p className="text-xs text-fg-muted">
                {dateFormat.format(new Date(entry.createdAt))} · {HISTORY_ACTION_LABELS[entry.action]}
              </p>
              <p className="mt-1 line-clamp-3 whitespace-pre-wrap text-xs text-fg">{entry.body}</p>
              <div className="mt-2 flex gap-2">
                <Button size="sm" variant="ghost" onClick={() => onLoad(entry)}>
                  편집기로 불러오기
                </Button>
                <Button size="sm" variant="secondary" loading={restoring} onClick={() => onRestore(entry)}>
                  이 버전으로 복원
                </Button>
              </div>
            </li>
          ))}
        </ul>
      )}
    </BottomSheet>
  );
}
