
import { Link } from "@/src/shared/lib/router";
import { useState, type FormEvent } from "react";
import { Button, EmptyState, ErrorState, Field, FieldTextarea, ListSkeleton } from "@/src/shared/ui/kit";
import { QUESTION_STATUS_LABELS, useAskQuestion, useQuestions } from "@/src/entities/punisher";
import { scrollToAnchor } from "@/src/features/punisher/lecture-progress";
import { describePunisherError, localizeServerText, type Question } from "@/src/shared/api/punisher";
import { InlineError } from "@/src/shared/ui/punisher";
import { Markdown } from "@/src/shared/ui/punisher/Markdown";
import { useT } from "@/src/shared/i18n";
import { messages } from "../i18n";

function jumpTo(anchor: string, onJump?: () => void) {
  onJump?.();
  requestAnimationFrame(() => scrollToAnchor(anchor));
}

function QuestionItem({ q, onJump }: { q: Question; onJump?: () => void }) {
  const t = useT(messages);
  const linked = q.linkedConceptIds ?? [];
  const proposed = q.proposedConcepts ?? [];
  return (
    <li className="flex flex-col gap-2 border-b border-line py-3 text-sm last:border-b-0">
      <div className="flex items-start justify-between gap-2">
        <p className="font-medium text-fg">
          {q.kind === "alt_explanation" ? t.altRequestPrefix : ""}
          {q.question}
        </p>
        <span className="shrink-0 text-xs text-fg-muted">{QUESTION_STATUS_LABELS[q.status]}</span>
      </div>
      {q.anchor && (
        <button
          type="button"
          onClick={() => jumpTo(q.anchor, onJump)}
          className="self-start text-xs text-fg-muted underline underline-offset-2 hover:text-fg"
        >
          {t.jumpToQuestion}
        </button>
      )}
      {q.status === "answered" && <Markdown className="text-sm leading-6">{q.answer}</Markdown>}
      {q.status === "failed" && <p className="text-danger">{localizeServerText(q.error, t.answerFailed)}</p>}
      {linked.length > 0 && (
        <p className="text-xs text-fg-secondary">
          {t.linkedConcepts}{" "}
          {linked.map((id, index) => (
            <span key={id}>
              {index > 0 && ", "}
              <Link className="text-accent-text underline" href={`/map?concept=${encodeURIComponent(id)}`}>
                {id}
              </Link>
            </span>
          ))}
        </p>
      )}
      {proposed.length > 0 && (
        <div className="border-l-2 border-line-strong pl-2 text-xs text-fg">
          <p className="font-semibold text-fg-secondary">{t.proposedConcepts}</p>
          <ul className="mt-1 list-disc pl-4">
            {proposed.map((p) => (
              <li key={p.id}>
                {p.name} <span className="text-fg-muted">({p.id})</span>
                {p.summary && ` · ${p.summary}`}
              </li>
            ))}
          </ul>
        </div>
      )}
    </li>
  );
}

interface QuestionsPanelProps {
  lectureId: string;
  getAnchor: () => string;
  /** "위치로 이동" 전에 시트 닫기 */
  onJump?: () => void;
}

export function QuestionsPanel({ lectureId, getAnchor, onJump }: QuestionsPanelProps) {
  const t = useT(messages);
  const list = useQuestions(lectureId);
  const ask = useAskQuestion(lectureId);
  const [text, setText] = useState("");

  function submit(event: FormEvent) {
    event.preventDefault();
    const question = text.trim();
    if (!question || ask.isPending) { return; }
    ask.mutate({ anchor: getAnchor(), question }, { onSuccess: () => setText("") });
  }

  const items = list.data ?? [];

  return (
    <div className="flex flex-col gap-3">
      <form onSubmit={submit} className="flex flex-col gap-2">
        <Field label={t.askLabel}>
          <FieldTextarea value={text} maxLength={2000} rows={3} onChange={(event) => setText(event.target.value)} />
        </Field>
        <Button type="submit" size="sm" variant="secondary" loading={ask.isPending} disabled={!text.trim()} className="self-end">
          {ask.isPending ? t.asking : t.ask}
        </Button>
        <div aria-live="polite">
          <InlineError error={ask.error} />
        </div>
      </form>
      {list.isLoading ? (
        <ListSkeleton rows={2} />
      ) : list.error ? (
        <ErrorState compact message={describePunisherError(list.error)} onRetry={() => list.refetch()} />
      ) : items.length === 0 ? (
        <EmptyState compact title={t.noQuestions} />
      ) : (
        <ul role="list">
          {[...items].reverse().map((q) => (
            <QuestionItem key={q.id} q={q} onJump={onJump} />
          ))}
        </ul>
      )}
    </div>
  );
}
