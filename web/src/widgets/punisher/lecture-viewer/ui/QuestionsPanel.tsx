
import { Link } from "@/src/shared/lib/router";
import { useState, type FormEvent } from "react";
import { Button, EmptyState, ErrorState, ListSkeleton, TextField } from "@forgen-team/ui";
import { QUESTION_STATUS_LABELS, useAskQuestion, useQuestions } from "@/src/entities/punisher";
import { scrollToAnchor } from "@/src/features/punisher/lecture-progress";
import { PassageThreadList, passageQuestionIds, useReadingThreads } from "@/src/features/punisher/reading-assist";
import { describePunisherError, localizeServerText, type Question } from "@/src/shared/api/punisher";
import { InlineError } from "@/src/shared/ui/punisher";
import { Markdown } from "@/src/shared/ui/punisher/Markdown";
import { useT } from "@/src/shared/i18n";
import { messages } from "../i18n";

type JumpHandler = (scroll: () => void) => void;

function jumpTo(anchor: string, onJump?: JumpHandler) {
  const scroll = () => scrollToAnchor(anchor);
  if (onJump) { onJump(scroll); } else { requestAnimationFrame(scroll); }
}

function QuestionItem({ q, onJump }: { q: Question; onJump?: JumpHandler }) {
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
              <Link className="text-accent-text underline" href={`/map?node=${encodeURIComponent(id)}`}>
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
  /** "위치로 이동" — 시트를 닫고 닫힌 뒤 scroll 을 부른다(없으면 바로 스크롤) */
  onJump?: JumpHandler;
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

  const threads = useReadingThreads();
  // 구절 스레드(인용·이어 묻기)는 위 구절별 목록에서 보이고, 여기는 그 밖의 것만 남긴다 —
  // 예전 방식 질문과, 루트가 목록에 없어 구절에 묶이지 못한 이어 묻기(사라지지 않게).
  const inPassages = threads ? passageQuestionIds(threads.passages) : null;
  const items = (list.data ?? []).filter((q) => !inPassages || !inPassages.has(q.id));
  const hasPassages = (threads?.passages.length ?? 0) > 0 || threads?.focus?.kind === "draft";

  return (
    <div className="flex flex-col gap-3">
      <form onSubmit={submit} className="flex flex-col gap-2">
        <TextField
          multiline
          label={t.askLabel}
          value={text}
          onValueChange={({ value }) => setText(value)}
          hideCharacterCount
          textareaProps={{ rows: 3, maxLength: 2000 }}
        />
        <Button type="submit" size="sm" variant="secondary" loading={ask.isPending} disabled={!text.trim()} className="self-end">
          {ask.isPending ? t.asking : t.ask}
        </Button>
        <div aria-live="polite">
          <InlineError error={ask.error} />
        </div>
      </form>
      <PassageThreadList />
      {list.isLoading ? (
        <ListSkeleton rows={2} />
      ) : list.error ? (
        <ErrorState title={describePunisherError(list.error)} onRetry={() => void list.refetch()} />
      ) : items.length === 0 ? (
        hasPassages ? null : <EmptyState title={t.noQuestions} />
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
