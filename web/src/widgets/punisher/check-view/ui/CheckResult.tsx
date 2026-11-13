
import { Link } from "@/src/shared/lib/router";
import { forwardRef, useState, type ReactNode } from "react";
import { Button, Section } from "@/src/shared/ui/kit";
import { MasteryDots, describeCheckError, masteryLabel, useCompleteLecture, useRelearn } from "@/src/entities/punisher";
import { useLaunchReview } from "@/src/features/punisher/check-start";
import { InlineError } from "@/src/shared/ui/punisher";
import type { ResultRow } from "../lib/checkView";

const reviewDate = new Intl.DateTimeFormat("ko-KR", { month: "long", day: "numeric", weekday: "short" });

/** 반복 실패(lapses ≥ 2)한 목표에 '다시 배우기'를 제안한다(design §2.4). */
const RELEARN_LAPSES = 2;

interface CheckResultProps {
  rows: ResultRow[];
  /** 앵커 → '다시 볼 곳' 링크·글자 */
  anchorLink: (anchor: string) => { href: string; label: string } | null;
  /** 아래 동작(확인: 강의 완료로 표시 · 복습: 오늘 더 하기) */
  children?: ReactNode;
}

/** 확인 결과의 '강의 완료로 표시'. */
export function LectureCompleteAction({ lectureId, lectureCompleted }: { lectureId: string; lectureCompleted: boolean }) {
  const complete = useCompleteLecture(lectureId);
  const completed = lectureCompleted || complete.isSuccess;
  return (
    <>
      <Button onClick={() => complete.mutate()} disabled={completed} loading={complete.isPending}>
        {completed ? "강의 완료함" : "강의 완료로 표시"}
      </Button>
      <div aria-live="polite" className="w-full empty:hidden">
        <InlineError error={complete.error} />
      </div>
    </>
  );
}

/** 복습 결과의 '오늘 더 하기' — 만기된 복습이 남아 있을 때만(POST /reviews 를 다시 부른다). */
export function ReviewMoreAction({ remaining }: { remaining: number }) {
  const review = useLaunchReview();
  if (remaining <= 0) { return null; }
  return (
    <>
      <Button onClick={review.launch} loading={review.isPending}>
        오늘 더 하기
      </Button>
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

/** 결과 요약(design §4.4): 목표별 레벨 변화 · 놓친 기준 한 줄 · 다시 볼 곳 · 드러난 오개념 · 다음 복습일 · 다시 배우기 + 아래 동작. */
export const CheckResult = forwardRef<HTMLElement, CheckResultProps>(function CheckResult(
  { rows, anchorLink, children },
  ref,
) {
  const relearn = useRelearn();
  const [relearnedIds, setRelearnedIds] = useState<ReadonlySet<string>>(() => new Set());
  const runRelearn = (objectiveId: string) =>
    relearn.mutate(objectiveId, { onSuccess: () => setRelearnedIds((prev) => new Set(prev).add(objectiveId)) });
  return (
    <Section ref={ref} tabIndex={-1} title="결과" className="mt-8 border-t border-line pt-4 outline-none">
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
              {row.misconception && <p className="break-keep text-fg-secondary">드러난 오개념: {row.misconception}</p>}
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
              {row.lapses >= RELEARN_LAPSES && (
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
