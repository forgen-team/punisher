
import { Link } from "@/src/shared/lib/router";
import { forwardRef, useState, type ReactNode } from "react";
import { Button, Section, buttonVariants } from "@/src/shared/ui/kit";
import { MasteryDots, describeCheckError, masteryLabel, useCompleteLecture, useRelearn } from "@/src/entities/punisher";
import { useLaunchReview } from "@/src/features/punisher/check-start";
import { InlineError } from "@/src/shared/ui/punisher";
import { DIAGNOSTIC_LABELS } from "@/src/features/punisher/pretest";
import { showPrediction, suggestRelearn, type ResultRow } from "../lib/checkView";
import { PREDICTION_LABELS, RESULT_ACTION_LABELS } from "./constants";
import { LectureSummary } from "./LectureSummary";
import { MisconceptionList } from "./MisconceptionList";

const TODAY_HREF = "/";

const reviewDate = new Intl.DateTimeFormat("ko-KR", { month: "long", day: "numeric", weekday: "short" });

interface CheckResultProps {
  checkId: string;
  rows: ResultRow[];
  /** 이 세션에서 오개념을 인정한 턴 */
  acknowledgedTurnIds: ReadonlySet<string>;
  /** 앵커 → '다시 볼 곳' 링크·글자 */
  anchorLink: (anchor: string) => { href: string; label: string } | null;
  /** 확인: 그 강의의 한 장 요약(없으면 생략) */
  lectureSummary?: string;
  /** 아래 동작(확인: 강의 완료로 표시 · 복습: 오늘 더 하기) */
  children?: ReactNode;
}

/**
 * 확인 결과의 다음 행동 — 강의를 아직 완료하지 않았으면 '강의 완료로 표시', 늘 '오늘로'
 * (완료한 강의면 '오늘로'가 주 버튼 — 눌리지 않는 버튼으로 끝나지 않게).
 */
export function LectureCompleteAction({ lectureId, lectureCompleted }: { lectureId: string; lectureCompleted: boolean }) {
  const complete = useCompleteLecture(lectureId);
  const completed = lectureCompleted || complete.isSuccess;
  return (
    <>
      {!completed && (
        <Button onClick={() => complete.mutate()} loading={complete.isPending}>
          강의 완료로 표시
        </Button>
      )}
      <Link href={TODAY_HREF} className={buttonVariants({ variant: completed ? "primary" : "secondary" })}>
        {RESULT_ACTION_LABELS.today}
      </Link>
      <div aria-live="polite" className="w-full empty:hidden">
        <InlineError error={complete.error} />
      </div>
    </>
  );
}

/** 복습 결과의 다음 행동 — 만기된 복습이 남아 있으면 '오늘 더 하기'(POST /reviews 를 다시 부른다), 늘 '오늘로'. */
export function ReviewMoreAction({ remaining }: { remaining: number }) {
  const review = useLaunchReview();
  return (
    <>
      {remaining > 0 && (
        <Button onClick={review.launch} loading={review.isPending}>
          오늘 더 하기
        </Button>
      )}
      <Link href={TODAY_HREF} className={buttonVariants({ variant: remaining > 0 ? "secondary" : "primary" })}>
        {RESULT_ACTION_LABELS.today}
      </Link>
      <div aria-live="polite" className="w-full empty:hidden">
        {review.error && (
          <p role="alert" className="text-sm text-danger">
            {describeCheckError(review.error)}
          </p>
        )}
      </div>
    </>
  );
}

function NextReview({ row }: { row: ResultRow }) {
  if (row.queued) { return <>복습이 밀려 대기</>; }
  if (!row.nextReviewAt) { return null; }
  return <>다음 복습 {reviewDate.format(new Date(row.nextReviewAt))}</>;
}

/** 결과 요약(design §4.4, 끝난 세션의 맨 위): 목표별 레벨 변화 · 놓친 기준 한 줄 · 다시 볼 곳 · 드러난 오개념 · 다음 복습일 · 다시 배우기 + 아래 동작. */
export const CheckResult = forwardRef<HTMLElement, CheckResultProps>(function CheckResult(
  { checkId, rows, acknowledgedTurnIds, anchorLink, lectureSummary, children },
  ref,
) {
  const relearn = useRelearn();
  const [relearnedIds, setRelearnedIds] = useState<ReadonlySet<string>>(() => new Set());
  const runRelearn = (objectiveId: string) =>
    relearn.mutate(objectiveId, { onSuccess: () => setRelearnedIds((prev) => new Set(prev).add(objectiveId)) });
  return (
    <Section ref={ref} tabIndex={-1} title="결과" className="mt-4 scroll-mt-28 border-b border-line pb-4 outline-none">
      <ul className="flex flex-col divide-y divide-line">
        {rows.map((row) => {
          const link = row.anchor ? anchorLink(row.anchor) : null;
          const missing = (row.missing ?? []).join(" · ");
          const relearned = relearnedIds.has(row.objectiveId);
          return (
            <li key={row.itemId} className="flex flex-col gap-1.5 py-3 text-sm">
              <p className="break-keep text-fg">{row.text}</p>
              <p className="flex flex-wrap items-center gap-x-2 gap-y-1 text-fg-secondary">
                <MasteryDots level={row.levelAfter} />
                <span>
                  {masteryLabel(row.levelBefore)} → {masteryLabel(row.levelAfter)}
                </span>
              </p>
              {missing && (
                <p className="truncate text-fg-secondary" title={missing}>
                  놓친 기준: {missing}
                </p>
              )}
              <MisconceptionList checkId={checkId} misconceptions={row.misconceptions} acknowledged={acknowledgedTurnIds} />
              {row.prediction && showPrediction(row) && (
                <p className="break-keep text-fg-secondary">
                  {PREDICTION_LABELS.before}: {row.prediction.dontKnow ? DIAGNOSTIC_LABELS.dontKnow : row.prediction.answer} →{" "}
                  {PREDICTION_LABELS.now}: {row.answer}
                </p>
              )}
              <div className="flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-fg-muted">
                <span>
                  <NextReview row={row} />
                </span>
                {link && (
                  <Link href={link.href} className="text-accent-text underline underline-offset-2">
                    다시 볼 곳: {link.label}
                  </Link>
                )}
              </div>
              {suggestRelearn(row) && (
                <div>
                  <Button
                    size="sm"
                    variant="secondary"
                    disabled={relearned}
                    loading={relearn.isPending && relearn.variables === row.objectiveId}
                    onClick={() => runRelearn(row.objectiveId)}
                  >
                    {relearned ? "다시 배우기로 바꿈" : "다시 배우기"}
                  </Button>
                </div>
              )}
            </li>
          );
        })}
      </ul>
      {lectureSummary && (
        <div className="mt-2">
          <LectureSummary text={lectureSummary} />
        </div>
      )}
      <div className="mt-4 flex flex-wrap items-center gap-2 empty:hidden">{children}</div>
      <div aria-live="polite" className="mt-2 empty:hidden">
        {relearn.error && (
          <p role="alert" className="text-sm text-danger">
            {describeCheckError(relearn.error)}
          </p>
        )}
      </div>
    </Section>
  );
});
