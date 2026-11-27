
import { Button, EmptyState, ErrorState, ListSkeleton, Sheet } from "@forgen-team/ui";
import { usePromptHistory } from "@/src/entities/punisher";
import { describePunisherError, type PromptHistoryEntry } from "@/src/shared/api/punisher";
import { formatDate, useLocale, translate } from "@/src/shared/i18n";
import { messages } from "../i18n";

const DATE_OPTIONS: Intl.DateTimeFormatOptions = { dateStyle: "medium", timeStyle: "short" };

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
  const locale = useLocale();
  const t = translate(messages, locale);
  const actionLabels = { save: t.historySave, reset: t.historyReset, restore: t.historyRestore };
  return (
    <Sheet open={open} onOpenChange={(o) => { if (!o) { onClose(); } }} title={t.history} description={t.historyDescription} desktop="side">
      {history.isLoading ? (
        <ListSkeleton rows={3} />
      ) : history.error ? (
        <ErrorState title={describePunisherError(history.error)} onRetry={() => { void history.refetch(); }} />
      ) : items.length === 0 ? (
        <EmptyState title={t.historyEmpty} />
      ) : (
        <ul role="list" className="flex flex-col gap-3">
          {items.map((entry) => (
            <li key={entry.id} className="rounded-md border border-line p-3">
              <p className="text-xs text-fg-muted">
                {formatDate(new Date(entry.createdAt), locale, DATE_OPTIONS)} · {actionLabels[entry.action]}
              </p>
              <p className="mt-1 line-clamp-3 whitespace-pre-wrap text-xs text-fg">{entry.body}</p>
              <div className="mt-2 flex flex-wrap gap-2">
                <Button size="sm" variant="ghost" onClick={() => onLoad(entry)}>
                  {t.loadIntoEditor}
                </Button>
                <Button size="sm" variant="secondary" loading={restoring} onClick={() => onRestore(entry)}>
                  {t.restoreThis}
                </Button>
              </div>
            </li>
          ))}
        </ul>
      )}
    </Sheet>
  );
}
