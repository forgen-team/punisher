
import { Link, useRouter } from "@/src/shared/lib/router";
import { forwardRef, useState, type ReactNode } from "react";
import { Button, Section } from "@forgen-team/ui";
import { MasteryDots, describeCheckError, masteryLabel, useCompleteLecture, useRelearn, useStartCheck, useToday } from "@/src/entities/punisher";
import { checkHref } from "@/src/features/punisher/check-start";
import { InlineError } from "@/src/shared/ui/punisher";
import { DIAGNOSTIC_MESSAGES } from "@/src/features/punisher/pretest";
import { formatDate, useLocale, useT } from "@/src/shared/i18n";
import { SYNTHESIS_RESULT_LABELS, messages } from "../i18n";
import { confidentErrors, showPrediction, suggestRelearn, type ResultRow, type ResultTally } from "../lib/checkView";
import type { SynthesisRow } from "../lib/synthesis";
import { LectureSummary } from "./LectureSummary";
import { MisconceptionList } from "./MisconceptionList";

const TODAY_HREF = "/";

const REVIEW_DATE: Intl.DateTimeFormatOptions = { month: "long", day: "numeric", weekday: "short" };
const HEADLINE_DATE: Intl.DateTimeFormatOptions = { month: "long", day: "numeric" };

interface CheckResultProps {
  checkId: string;
  /** 끝 화면 머리 — "복습 12개를 마쳤어요" */
  headline: string;
  /** 가장 이른 다음 복습일(ISO, 없으면 null) */
  nextReviewAt: string | null;
  /** 집계 "맞힘 n · 다시 볼 것 n · 건너뜀 n"(건너뜀은 있을 때만) */
  tally?: ResultTally;
  rows: ResultRow[];
  /** 이 세션에서 오개념을 인정한 턴 */
  acknowledgedTurnIds: ReadonlySet<string>;
  /** 앵커 → '다시 볼 곳' 링크·글자 */
  anchorLink: (anchor: string) => { href: string; label: string } | null;
  /** 확인: 그 강의의 한 장 요약(없으면 생략) */
  lectureSummary?: string;
  /** 종합 문제 결과 줄(v3 §5) */
  synthesis?: SynthesisRow[];
  /** 다음 행동(복습: 확인 대기 이어서 또는 오늘로 · 확인: 강의 완료로 표시 + 오늘로) */
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

/**
 * 복습 결과의 다음 행동 1개 — 확인 대기(완료했지만 확인하기를 안 한 강의)가 남았으면 그 확인 세션을 시작해(replace) 이어 가고,
 * 없으면 '오늘로'. 오늘 응답을 아직 받는 동안은 '오늘로'를 보인다.
 */
export function ReviewNextAction() {
  const t = useT(messages);
  const router = useRouter();
  const today = useToday();
  const start = useStartCheck();
  const unchecked = today.data?.unchecked ?? [];
  const first = unchecked[0];
  if (!first) {
    return (
      <Button asChild variant="primary">
        <Link href={TODAY_HREF}>{t.today}</Link>
      </Button>
    );
  }
  return (
    <>
      <Button
        variant="primary"
        loading={start.isPending}
        onClick={() => start.mutate(first.lectureId, { onSuccess: (detail) => router.replace(checkHref(detail.session.id)) })}
      >
        {t.continueUnchecked(unchecked.length)}
      </Button>
      <div aria-live="polite" className="w-full empty:hidden">
        {start.error && (
          <p role="alert" className="text-sm text-danger">
            {describeCheckError(start.error)}
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
  { checkId, headline, nextReviewAt, tally, rows, acknowledgedTurnIds, anchorLink, lectureSummary, synthesis, children },
  ref,
) {
  const locale = useLocale();
  const t = useT(messages);
  const dt = useT(DIAGNOSTIC_MESSAGES);
  const relearn = useRelearn();
  const [relearnedIds, setRelearnedIds] = useState<ReadonlySet<string>>(() => new Set());
  const runRelearn = (objectiveId: string) =>
    relearn.mutate(objectiveId, { onSuccess: () => setRelearnedIds((prev) => new Set(prev).add(objectiveId)) });
  return (
    <div ref={ref} tabIndex={-1} className="mt-4 scroll-mt-28 outline-none">
    <div className="flex flex-col gap-1 pb-4">
      <p className="text-xl font-semibold text-fg">{headline}</p>
      {tally && <p className="text-sm text-fg-secondary tabular-nums">{t.tally(tally.correct, tally.again, tally.skipped)}</p>}
      {nextReviewAt && <p className="text-sm text-fg-secondary">{t.nextReview(formatDate(new Date(nextReviewAt), locale, HEADLINE_DATE))}</p>}
      <div className="mt-3 flex flex-wrap items-center gap-2 empty:hidden">{children}</div>
    </div>
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
                {row.outcome === "skipped" && <span>· {t.skippedRow}</span>}
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
