
import { useState, type FormEvent } from "react";
import { Trash2 } from "lucide-react";
import { Button, EmptyState, ErrorState, Field, FieldTextarea, IconButton, ListSkeleton } from "@/src/shared/ui/kit";
import { useCreateNote, useDeleteNote, useNotes } from "@/src/entities/punisher";
import { scrollToAnchor } from "@/src/features/punisher/lecture-progress";
import { describePunisherError } from "@/src/shared/api/punisher";
import { InlineError } from "@/src/shared/ui/punisher";

interface NotesPanelProps {
  lectureId: string;
  conceptId: string;
  getAnchor: () => string;
  /** "위치로 이동" 전에 시트 닫기 */
  onJump?: () => void;
}

/** 앵커 기준 메모 (design §3.3 메모·하이라이트). */
export function NotesPanel({ lectureId, conceptId, getAnchor, onJump }: NotesPanelProps) {
  const notes = useNotes({ lectureId });
  const create = useCreateNote();
  const remove = useDeleteNote();
  const [body, setBody] = useState("");

  function submit(event: FormEvent) {
    event.preventDefault();
    const text = body.trim();
    if (!text || create.isPending) { return; }
    create.mutate({ lectureId, conceptId, anchor: getAnchor(), kind: "note", body: text }, { onSuccess: () => setBody("") });
  }

  function jumpTo(anchor: string) {
    onJump?.();
    requestAnimationFrame(() => scrollToAnchor(anchor));
  }

  const items = notes.data ?? [];

  return (
    <div className="flex flex-col gap-3">
      <form onSubmit={submit} className="flex flex-col gap-2">
        <Field label="현재 위치에 메모">
          <FieldTextarea value={body} rows={3} maxLength={10000} onChange={(event) => setBody(event.target.value)} />
        </Field>
        <Button type="submit" size="sm" variant="secondary" loading={create.isPending} disabled={!body.trim()} className="self-end">
          메모 저장
        </Button>
        <div aria-live="polite">
          <InlineError error={create.error ?? remove.error} />
        </div>
      </form>
      {notes.isLoading ? (
        <ListSkeleton rows={2} />
      ) : notes.error ? (
        <ErrorState compact message={describePunisherError(notes.error)} onRetry={() => notes.refetch()} />
      ) : items.length === 0 ? (
        <EmptyState compact title="메모가 없습니다" />
      ) : (
        <ul role="list">
          {items.map((n) => (
            <li key={n.id} className="flex items-start gap-2 border-b border-line py-3 text-sm last:border-b-0">
              <div className="min-w-0 flex-1">
                {n.quote && <blockquote className="mb-1 border-l-2 border-line-strong pl-2 text-fg-secondary">{n.quote}</blockquote>}
                <p className="whitespace-pre-wrap break-words text-fg">{n.body}</p>
                {n.anchor && (
                  <button
                    type="button"
                    className="mt-1 text-xs text-fg-muted underline underline-offset-2 hover:text-fg"
                    onClick={() => jumpTo(n.anchor)}
                  >
                    메모 위치로 이동
                  </button>
                )}
              </div>
              <IconButton aria-label="메모 삭제" size="sm" disabled={remove.isPending} onClick={() => remove.mutate(n.id)}>
                <Trash2 aria-hidden="true" />
              </IconButton>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
