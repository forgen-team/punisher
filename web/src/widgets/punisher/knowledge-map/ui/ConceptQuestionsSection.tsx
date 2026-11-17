
import { Link } from "@/src/shared/lib/router";
import { MessageSquare } from "lucide-react";
import { ErrorState, List, ListRow, ListSkeleton, Section } from "@/src/shared/ui/kit";
import { INQUIRY_STATUS_LABELS, InquiryStatusIcon, QUESTION_STATUS_LABELS, inquiryRowTitle, useInquiries, useMyQuestions } from "@/src/entities/punisher";
import { inquiryHref, lectureHref } from "@/src/features/punisher/instant-start";
import { describePunisherError } from "@/src/shared/api/punisher";
import { useT } from "@/src/shared/i18n";
import { messages } from "../i18n";

/**
 * 이 개념에 대한 내 질문 — '이 질문에 답하기' 해설(해설에 이 개념이 나온 것, 다 본 것 포함)과
 * 강의 안에서 한 질문(최신 20개). 해설은 관련 개념 노드마다 붙어서 나중에 개념을 볼 때 함께 보인다.
 */
function QuestionRows({ conceptId }: { conceptId: string }) {
  const questions = useMyQuestions(conceptId);
  const t = useT(messages);
  const inquiries = useInquiries({ concept: conceptId });
  const items = questions.data?.pages.flatMap((p) => p.items) ?? [];
  const asked = inquiries.data ?? [];
  if (questions.isLoading || inquiries.isLoading) { return <ListSkeleton rows={1} />; }
  const error = questions.error ?? inquiries.error;
  if (error) {
    return (
      <ErrorState
        compact
        message={describePunisherError(error)}
        onRetry={() => {
          void questions.refetch();
          void inquiries.refetch();
        }}
      />
    );
  }
  if (items.length === 0 && asked.length === 0) { return <p className="text-xs text-fg-muted">{t.questionsEmpty}</p>; }
  return (
    <List>
      {asked.map((q) => (
        <ListRow
          key={q.id}
          href={inquiryHref(q.id)}
          linkAs={Link}
          leading={<InquiryStatusIcon status={q.status} />}
          title={inquiryRowTitle(q)}
          meta={t.inquiryMeta(INQUIRY_STATUS_LABELS[q.status] ?? q.status)}
        />
      ))}
      {items.map((q) =>
        q.lectureId ? (
          <ListRow
            key={q.id}
            href={`${lectureHref(q.lectureId)}${q.anchor ? `#pa-${q.anchor}` : ""}`}
            linkAs={Link}
            leading={<MessageSquare aria-hidden="true" />}
            title={q.question}
            meta={t.lectureQuestionMeta(QUESTION_STATUS_LABELS[q.status])}
          />
        ) : (
          <ListRow key={q.id} leading={<MessageSquare aria-hidden="true" />} title={q.question} meta={QUESTION_STATUS_LABELS[q.status]} />
        ),
      )}
    </List>
  );
}

export function ConceptQuestionsSection({ conceptId }: { conceptId: string }) {
  const t = useT(messages);
  return (
    <Section title={t.questions} headingLevel={3}>
      <QuestionRows conceptId={conceptId} />
    </Section>
  );
}
