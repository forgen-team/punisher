
import { Link } from "@/src/shared/lib/router";
import { useState, type FormEvent } from "react";
import { Button, EmptyState, ErrorState, Field, FieldTextarea, ListSkeleton } from "@/src/shared/ui/kit";
import { QUESTION_STATUS_LABELS, useAskQuestion, useQuestions } from "@/src/entities/punisher";
import { scrollToAnchor } from "@/src/features/punisher/lecture-progress";
import { describePunisherError, type Question } from "@/src/shared/api/punisher";
import { InlineError } from "@/src/shared/ui/punisher";
import { Markdown } from "@/src/shared/ui/punisher/Markdown";

function jumpTo(anchor: string, onJump?: () => void) {
  onJump?.();
  requestAnimationFrame(() => scrollToAnchor(anchor));
}

function QuestionItem({ q, onJump }: { q: Question; onJump?: () => void }) {
  const linked = q.linkedConceptIds ?? [];
  const proposed = q.proposedConcepts ?? [];
  return (
    <li className="flex flex-col gap-2 border-b border-line py-3 text-sm last:border-b-0">
      <div className="flex items-start justify-between gap-2">
        <p className="font-medium text-fg">
          {q.kind === "alt_explanation" ? "재설명 요청: " : ""}
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
          질문한 위치로 이동
        </button>
      )}
      {q.status === "answered" && <Markdown className="text-sm leading-6">{q.answer}</Markdown>}
      {q.status === "failed" && <p className="text-danger">{q.error || "답변 생성에 실패했습니다."}</p>}
      {linked.length > 0 && (
        <p className="text-xs text-fg-secondary">
          연결된 개념:{" "}
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
          <p className="font-semibold text-fg-secondary">그래프에 추가 제안된 개념</p>
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
        <Field label="지금 읽는 부분에 대해 질문" hint="섹션 문맥과 내 학습 이력을 함께 보내 답변합니다">
          <FieldTextarea value={text} maxLength={2000} rows={3} onChange={(event) => setText(event.target.value)} />
        </Field>
        <Button type="submit" size="sm" variant="secondary" loading={ask.isPending} disabled={!text.trim()} className="self-end">
          {ask.isPending ? "답변 생성 중" : "질문하기"}
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
        <EmptyState compact title="아직 질문이 없습니다" />
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
