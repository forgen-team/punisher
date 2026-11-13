
import { CheckCircle2 } from "lucide-react";
import { Button, Section } from "@/src/shared/ui/kit";
import { describeCheckError } from "@/src/entities/punisher";
import { useLaunchReview } from "@/src/features/punisher/check-start";
import type { ReviewCardState } from "../lib/reviewCard";

type VisibleState = Exclude<ReviewCardState, { kind: "hidden" }>;

/** 오늘 복습 카드 — "복습 n개 · 약 m분", 시간 합계(세션 합계가 따로 보이면 생략), '복습 시작'/'오늘 더 하기'. 마친 날은 한 줄로 접는다. */
export function ReviewCard({ state, showTotal = true }: { state: VisibleState; showTotal?: boolean }) {
  const review = useLaunchReview();
  if (state.kind === "done") {
    return (
      <Section variant="card" aria-label="복습">
        <p className="flex min-h-touch items-center gap-2 text-sm text-fg-secondary">
          <CheckCircle2 className="size-4 shrink-0 text-success" aria-hidden="true" />
          오늘 복습 완료 · {state.done}/{state.total}
        </p>
      </Section>
    );
  }
  return (
    <Section variant="card" title="복습">
      <div className="flex flex-wrap items-center justify-between gap-3 pb-4 pt-1">
        <div className="flex min-w-0 flex-col gap-0.5">
          <p className="text-sm font-medium text-fg">
            복습 {state.due}개 · 약 {state.minutes}분
          </p>
          {showTotal && (
            <p className="text-xs text-fg-muted">
              복습 {state.minutes}분 + 강의 {state.lectureMinutes}분
            </p>
          )}
          {state.backlog !== null && <p className="text-xs text-warning">밀린 복습 {state.backlog}개</p>}
        </div>
        <Button size="sm" onClick={review.launch} loading={review.isPending}>
          {state.action === "more" ? "오늘 더 하기" : "복습 시작"}
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
