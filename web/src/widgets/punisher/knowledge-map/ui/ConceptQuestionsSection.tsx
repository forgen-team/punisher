
import { Link } from "@/src/shared/lib/router";
import { ErrorState, List, ListRow, ListSkeleton, Section } from "@/src/shared/ui/kit";
import { QUESTION_STATUS_LABELS, useMyQuestions } from "@/src/entities/punisher";
import { lectureHref } from "@/src/features/punisher/instant-start";
import { describePunisherError } from "@/src/shared/api/punisher";

function QuestionRows({ conceptId }: { conceptId: string }) {
  const questions = useMyQuestions(conceptId);
  const items = questions.data?.pages.flatMap((p) => p.items) ?? [];
  if (questions.isLoading) { return <ListSkeleton rows={1} />; }
  if (questions.error) {
    return <ErrorState compact message={describePunisherError(questions.error)} onRetry={() => questions.refetch()} />;
  }
  if (items.length === 0) { return <p className="text-xs text-fg-muted">질문이 없습니다.</p>; }
  return (
    <List>
      {items.map((q) =>
        q.lectureId ? (
          <ListRow
            key={q.id}
            href={`${lectureHref(q.lectureId)}${q.anchor ? `#pa-${q.anchor}` : ""}`}
            linkAs={Link}
            title={q.question}
            meta={QUESTION_STATUS_LABELS[q.status]}
          />
        ) : (
          <ListRow key={q.id} title={q.question} meta={QUESTION_STATUS_LABELS[q.status]} />
        ),
      )}
    </List>
  );
}

/** 이 개념과 관련해 내가 한 질문(최신 20개). */
export function ConceptQuestionsSection({ conceptId }: { conceptId: string }) {
  return (
    <Section title="내 질문" headingLevel={3}>
      <QuestionRows conceptId={conceptId} />
    </Section>
  );
}
