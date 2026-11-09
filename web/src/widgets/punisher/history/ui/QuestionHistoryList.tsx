
import { Link } from "@/src/shared/lib/router";
import { useState } from "react";
import { BottomSheet, EmptyState, ErrorState, List, ListRow, ListSkeleton, buttonVariants } from "@/src/shared/ui/kit";
import { QUESTION_STATUS_LABELS, useMyQuestions } from "@/src/entities/punisher";
import { lectureHref } from "@/src/features/punisher/instant-start";
import { describePunisherError, type MyQuestion } from "@/src/shared/api/punisher";
import { Markdown } from "@/src/shared/ui/punisher/Markdown";
import { LoadMore } from "./LoadMore";

const dateFormat = new Intl.DateTimeFormat("ko-KR", { dateStyle: "medium" });

function metaOf(q: MyQuestion): string {
  return [q.conceptName, QUESTION_STATUS_LABELS[q.status], dateFormat.format(new Date(q.createdAt))].filter(Boolean).join(" · ");
}

function QuestionSheet({ question, onClose }: { question: MyQuestion | null; onClose: () => void }) {
  return (
    <BottomSheet
      open={Boolean(question)}
      onClose={onClose}
      title={question?.question ?? ""}
      description={question ? metaOf(question) : undefined}
      desktop="side"
      footer={
        question?.lectureId ? (
          <Link
            href={`${lectureHref(question.lectureId)}${question.anchor ? `#pa-${question.anchor}` : ""}`}
            className={buttonVariants({ variant: "secondary" })}
          >
            {question.conceptName ? `강의에서 보기: ${question.conceptName}` : "강의에서 보기"}
          </Link>
        ) : null
      }
    >
      {question?.status === "answered" && question.answer ? (
        <Markdown className="text-base leading-[1.7]">{question.answer}</Markdown>
      ) : question?.status === "failed" ? (
        <p className="text-sm text-danger">{question.error || "답변 생성에 실패했습니다."}</p>
      ) : (
        <p className="text-sm text-fg-muted">아직 답변이 없습니다.</p>
      )}
    </BottomSheet>
  );
}

/** 강의 전체에서 내가 한 질문(최신순). 누르면 답변 전문 시트. */
export function QuestionHistoryList() {
  const questions = useMyQuestions();
  const [open, setOpen] = useState<MyQuestion | null>(null);
  const items = questions.data?.pages.flatMap((p) => p.items) ?? [];
  if (questions.isLoading) { return <ListSkeleton rows={5} />; }
  if (questions.error) {
    return <ErrorState message={describePunisherError(questions.error)} onRetry={() => questions.refetch()} />;
  }
  if (items.length === 0) {
    return <EmptyState title="질문 기록이 없습니다" description="강의 화면의 질문과 메모에서 질문할 수 있습니다." />;
  }
  return (
    <div className="flex flex-col gap-3">
      <List>
        {items.map((q) => (
          <ListRow key={q.id} onClick={() => setOpen(q)} title={q.question} meta={metaOf(q)} />
        ))}
      </List>
      <LoadMore hasNextPage={Boolean(questions.hasNextPage)} isFetching={questions.isFetchingNextPage} onLoad={() => questions.fetchNextPage()} />
      <QuestionSheet question={open} onClose={() => setOpen(null)} />
    </div>
  );
}
