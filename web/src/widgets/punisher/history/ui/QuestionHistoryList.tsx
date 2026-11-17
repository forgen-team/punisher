
import { Link } from "@/src/shared/lib/router";
import { useState } from "react";
import { BottomSheet, EmptyState, ErrorState, List, ListRow, ListSkeleton, buttonVariants } from "@/src/shared/ui/kit";
import { QUESTION_STATUS_LABELS_BY_LOCALE, useMyQuestions } from "@/src/entities/punisher";
import { lectureHref } from "@/src/features/punisher/instant-start";
import { describePunisherError, type MyQuestion } from "@/src/shared/api/punisher";
import { useWideLayout } from "@/src/shared/lib/useWideLayout";
import { Markdown } from "@/src/shared/ui/punisher/Markdown";
import { LoadMore } from "./LoadMore";
import { useLocale, useT, type Locale } from "@/src/shared/i18n";
import { historyDay } from "../lib/historyDate";
import { messages } from "../i18n";


function metaOf(q: MyQuestion, locale: Locale): string {
  return [q.conceptName, QUESTION_STATUS_LABELS_BY_LOCALE[locale][q.status], historyDay(q.createdAt, locale)].filter(Boolean).join(" · ");
}

function QuestionAnswer({ question }: { question: MyQuestion }) {
  const t = useT(messages);
  if (question.status === "answered" && question.answer) {
    return <Markdown className="text-base leading-[1.7]">{question.answer}</Markdown>;
  }
  if (question.status === "failed") {
    return <p className="text-sm text-danger">{question.error || t.answerFailed}</p>;
  }
  return <p className="text-sm text-fg-muted">{t.noAnswer}</p>;
}

function LectureLink({ question }: { question: MyQuestion }) {
  const t = useT(messages);
  if (!question.lectureId) { return null; }
  return (
    <Link
      href={`${lectureHref(question.lectureId)}${question.anchor ? `#pa-${question.anchor}` : ""}`}
      className={buttonVariants({ variant: "secondary" })}
    >
      {question.conceptName ? t.viewInLectureNamed(question.conceptName) : t.viewInLecture}
    </Link>
  );
}

function QuestionSheet({ question, onClose }: { question: MyQuestion | null; onClose: () => void }) {
  const locale = useLocale();
  return (
    <BottomSheet
      open={Boolean(question)}
      onClose={onClose}
      title={question?.question ?? ""}
      description={question ? metaOf(question, locale) : undefined}
      desktop="side"
      footer={question ? <LectureLink question={question} /> : null}
    >
      {question && <QuestionAnswer question={question} />}
    </BottomSheet>
  );
}

/** 넓은 화면(xl) 오른쪽 열: 고른 질문의 답변 전문(시트 없이). */
function QuestionPanel({ question }: { question: MyQuestion }) {
  const locale = useLocale();
  return (
    <article aria-labelledby={`q-${question.id}`} className="flex flex-col gap-4 rounded-xl border border-line bg-canvas-elevated p-4 shadow-xs">
      <header className="flex flex-col gap-1">
        <h2 id={`q-${question.id}`} className="break-keep text-lg font-semibold text-fg">
          {question.question}
        </h2>
        <p className="text-xs text-fg-muted">{metaOf(question, locale)}</p>
      </header>
      <QuestionAnswer question={question} />
      {question.lectureId && (
        <div className="border-t border-line pt-4">
          <LectureLink question={question} />
        </div>
      )}
    </article>
  );
}

/**
 * 강의 전체에서 내가 한 질문(최신순). 누르면 답변 전문 — xl 미만은 시트, xl 이상은 목록 | 답변 두 열(첫 질문을 미리 펼침).
 */
export function QuestionHistoryList() {
  const locale = useLocale();
  const t = useT(messages);
  const questions = useMyQuestions();
  const wide = useWideLayout();
  const [openId, setOpenId] = useState<string | null>(null);
  const items = questions.data?.pages.flatMap((p) => p.items) ?? [];
  if (questions.isLoading) { return <ListSkeleton rows={5} />; }
  if (questions.error) {
    return <ErrorState message={describePunisherError(questions.error)} onRetry={() => questions.refetch()} />;
  }
  if (items.length === 0) {
    return <EmptyState title={t.noQuestions} description={t.noQuestionsDescription} />;
  }
  const open = items.find((q) => q.id === openId) ?? null;
  const shown = open ?? items[0];
  return (
    <div className="grid gap-6 xl:grid-cols-[minmax(0,1fr)_minmax(0,1.2fr)] xl:items-start">
      <div className="flex min-w-0 flex-col gap-3">
        <List>
          {items.map((q) => (
            <ListRow key={q.id} onClick={() => setOpenId(q.id)} selected={wide === true && q.id === shown.id} title={q.question} meta={metaOf(q, locale)} />
          ))}
        </List>
        <LoadMore hasNextPage={Boolean(questions.hasNextPage)} isFetching={questions.isFetchingNextPage} onLoad={() => questions.fetchNextPage()} />
      </div>
      <div className="hidden min-w-0 xl:sticky xl:top-[calc(var(--app-header-offset)+1rem)] xl:block xl:max-h-[calc(100dvh-var(--app-header-offset)-2rem)] xl:overflow-y-auto">
        {wide === true && <QuestionPanel question={shown} />}
      </div>
      <QuestionSheet question={wide === false ? open : null} onClose={() => setOpenId(null)} />
    </div>
  );
}
