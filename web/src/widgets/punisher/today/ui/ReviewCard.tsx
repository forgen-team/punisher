
import { CheckCircle2 } from "lucide-react";
import { Button, Section } from "@forgen-team/ui";
import { describeCheckError } from "@/src/entities/punisher";
import { useLaunchReview } from "@/src/features/punisher/check-start";
import { useT } from "@/src/shared/i18n";
import { messages } from "../i18n";
import type { ReviewCardState } from "../lib/reviewCard";

type VisibleState = Exclude<ReviewCardState, { kind: "hidden" }>;

/** 오늘 복습 카드 — "복습 n개 · 약 m분", 시간 합계(세션 합계가 따로 보이면 생략), '복습 시작'/'오늘 더 하기'. 마친 날은 한 줄로 접는다. */
export function ReviewCard({ state, showTotal = true }: { state: VisibleState; showTotal?: boolean }) {
  const t = useT(messages);
  const review = useLaunchReview();
  if (state.kind === "done") {
    return (
      <Section variant="card" aria-label={t.review}>
        <p className="flex min-h-touch items-center gap-2 text-sm text-fg-secondary">
          <CheckCircle2 className="size-4 shrink-0 text-success" aria-hidden="true" />
          {t.reviewDone(state.done, state.total)}
        </p>
      </Section>
    );
  }
  return (
    <Section variant="card" title={t.review}>
      <div className="flex flex-wrap items-center justify-between gap-3 pb-4 pt-1">
        <div className="flex min-w-0 flex-col gap-0.5">
          <p className="text-sm font-medium text-fg">
            {t.reviewDue(state.due, state.minutes)}
          </p>
          {showTotal && (
            <p className="text-xs text-fg-muted">
              {t.reviewSplit(state.minutes, state.lectureMinutes)}
            </p>
          )}
          {state.backlog !== null && <p className="text-xs text-warning">{t.backlog(state.backlog)}</p>}
        </div>
        <Button variant="primary" size="sm" onClick={review.launch} loading={review.isPending}>
          {state.action === "more" ? t.doMore : t.startReview}
        </Button>
      </div>
      {review.error && (
        <p role="alert" className="pb-3 text-sm text-danger">
          {describeCheckError(review.error)}
        </p>
      )}
    </Section>
  );
}
