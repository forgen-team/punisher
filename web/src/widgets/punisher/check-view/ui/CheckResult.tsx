
import { Link } from "@/src/shared/lib/router";
import { forwardRef, useState, type ReactNode } from "react";
import { Button, Section } from "@forgen-team/ui";
import { MasteryDots, describeCheckError, masteryLabel, useCompleteLecture, useRelearn } from "@/src/entities/punisher";
import { useLaunchReview } from "@/src/features/punisher/check-start";
import { InlineError } from "@/src/shared/ui/punisher";
import { DIAGNOSTIC_MESSAGES } from "@/src/features/punisher/pretest";
import { formatDate, useLocale, useT } from "@/src/shared/i18n";
import { SYNTHESIS_RESULT_LABELS, messages } from "../i18n";
import { confidentErrors, showPrediction, suggestRelearn, type ResultRow } from "../lib/checkView";
import type { SynthesisRow } from "../lib/synthesis";
import { LectureSummary } from "./LectureSummary";
import { MisconceptionList } from "./MisconceptionList";

const TODAY_HREF = "/";

const REVIEW_DATE: Intl.DateTimeFormatOptions = { month: "long", day: "numeric", weekday: "short" };

interface CheckResultProps {
  checkId: string;
  rows: ResultRow[];
  /** 이 세션에서 오개념을 인정한 턴 */
  acknowledgedTurnIds: ReadonlySet<string>;
  /** 앵커 → '다시 볼 곳' 링크·글자 */
  anchorLink: (anchor: string) => { href: string; label: string } | null;
  /** 확인: 그 강의의 한 장 요약(없으면 생략) */
  lectureSummary?: string;
  /** 종합 문제 결과 줄(v3 §5) */
  synthesis?: SynthesisRow[];
  /** 아래 동작(확인: 강의 완료로 표시 · 복습: 오늘 더 하기) */
  children?: ReactNode;
}

/** 종합 문제 결과 — 문항마다 한 줄. */
function SynthesisList({ rows }: { rows: SynthesisRow[] }) {
  const locale = useLocale();
  return (
    <ul className="flex flex-col divide-y divide-line">
      {rows.map((row) => (
        <li key={row.itemId} className="flex flex-col gap-1 py-3 text-sm">
          <p className="line-clamp-2 break-keep text-fg">{row.prompt}</p>
          <p className="text-fg-secondary">{SYNTHESIS_RESULT_LABELS[locale][row.result]}</p>
        </li>
      ))}
    </ul>
  );
}

/** '착각' — 확실하다고 했는데 틀린 목표(있을 때만, 목표 목록 위). */
function ConfidentErrors({ rows }: { rows: ResultRow[] }) {
  const t = useT(messages);
  const list = confidentErrors(rows);
  if (list.length === 0) { return null; }
  return (
    <aside className="mb-2 rounded-md border-l-2 border-warning bg-warning-subtle px-3 py-2 text-sm">
      <p className="font-semibold text-warning">{t.confidentErrors(list.length)}</p>
      <ul className="mt-1 list-disc pl-5 text-fg-secondary">
        {list.map((r) => (
          <li key={r.itemId} className="break-keep">{r.text}</li>
        ))}
      </ul>
    </aside>
  );
}

/** 종합 문제는 '오늘로'만. */
export function SynthesisDoneAction() {
  const t = useT(messages);
  return (
    <Button asChild variant="primary">
      <Link href={TODAY_HREF}>{t.today}</Link>
    </Button>
  );
}

/**
 * 확인 결과의 다음 행동 — 강의를 아직 완료하지 않았으면 '강의 완료로 표시', 늘 '오늘로'
 * (완료한 강의면 '오늘로'가 주 버튼 — 눌리지 않는 버튼으로 끝나지 않게).
 */
export function LectureCompleteAction({ lectureId, lectureCompleted }: { lectureId: string; lectureCompleted: boolean }) {
  const t = useT(messages);
  const complete = useCompleteLecture(lectureId);
  const completed = lectureCompleted || complete.isSuccess;
  return (
    <>
      {!completed && (
        <Button variant="primary" onClick={() => complete.mutate()} loading={complete.isPending}>
          {t.markComplete}
        </Button>
      )}
      <Button asChild variant={completed ? "primary" : "secondary"}>
        <Link href={TODAY_HREF}>{t.today}</Link>
      </Button>
      <div aria-live="polite" className="w-full empty:hidden">
        <InlineError error={complete.error} />
      </div>
    </>
  );
}

/** 복습 결과의 다음 행동 — 만기된 복습이 남아 있으면 '오늘 더 하기'(POST /reviews 를 다시 부른다), 늘 '오늘로'. */
export function ReviewMoreAction({ remaining }: { remaining: number }) {
  const t = useT(messages);
  const review = useLaunchReview();
  return (
    <>
      {remaining > 0 && (
        <Button variant="primary" onClick={review.launch} loading={review.isPending}>
          {t.doMore}
        </Button>
      )}
      <Button asChild variant={remaining > 0 ? "secondary" : "primary"}>
        <Link href={TODAY_HREF}>{t.today}</Link>
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
  const locale = useLocale();
  const t = useT(messages);
  if (row.queued) { return <>{t.queued}</>; }
  if (!row.nextReviewAt) { return null; }
  return <>{t.nextReview(formatDate(new Date(row.nextReviewAt), locale, REVIEW_DATE))}</>;
}

/** 결과 요약(design §4.4, 끝난 세션의 맨 위): 목표별 레벨 변화 · 놓친 기준 한 줄 · 다시 볼 곳 · 드러난 오개념 · 다음 복습일 · 다시 배우기 + 아래 동작. */
export const CheckResult = forwardRef<HTMLDivElement, CheckResultProps>(function CheckResult(
  { checkId, rows, acknowledgedTurnIds, anchorLink, lectureSummary, synthesis, children },
  ref,
) {
  const t = useT(messages);
  const dt = useT(DIAGNOSTIC_MESSAGES);
  const relearn = useRelearn();
  const [relearnedIds, setRelearnedIds] = useState<ReadonlySet<string>>(() => new Set());
  const runRelearn = (objectiveId: string) =>
    relearn.mutate(objectiveId, { onSuccess: () => setRelearnedIds((prev) => new Set(prev).add(objectiveId)) });
  return (
    <div ref={ref} tabIndex={-1} className="mt-4 scroll-mt-28 outline-none">
    <Section title={t.result} className="border-b border-line pb-4">
      {synthesis && synthesis.length > 0 && <SynthesisList rows={synthesis} />}
      <ConfidentErrors rows={rows} />
      <ul className="flex flex-col divide-y divide-line empty:hidden">
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
                  {t.missingLine(missing)}
                </p>
              )}
              <MisconceptionList checkId={checkId} misconceptions={row.misconceptions} acknowledged={acknowledgedTurnIds} />
              {row.prediction && showPrediction(row) && (
                <p className="break-keep text-fg-secondary">
                  {t.predictionBefore}: {row.prediction.dontKnow ? dt.dontKnow : row.prediction.answer} →{" "}
                  {t.predictionNow}: {row.answer}
                </p>
              )}
              <div className="flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-fg-muted">
                <span>
                  <NextReview row={row} />
                </span>
                {link && (
                  <Link href={link.href} className="text-accent-text underline underline-offset-2">
                    {t.revisitLine(link.label)}
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
                    {relearned ? t.relearned : t.relearn}
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
    </div>
  );
});
