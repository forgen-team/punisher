import { useCallback, useEffect, useMemo, useRef, useState, type ReactNode } from "react";
import { Button, EmptyState, ErrorState, ProgressCircle, Skeleton, useConfirm, useToast } from "@forgen-team/ui";
import { usePageActions, useScreenTitle, type PageAction } from "@/src/shared/lib/screen-chrome";
import {
  describeCheckError,
  lectureAnchorHref,
  lectureHref,
  useCheckAbandon,
  useCheckResume,
  useCheckDispute,
  useCheckRetry,
  useCheckSession,
  useLecture,
} from "@/src/entities/punisher";
import type { CheckDetail, CheckItem, LectureSection } from "@/src/shared/api/punisher";
import { DiagnosticAnswer, currentDiagnosticItem } from "@/src/features/punisher/pretest";
import {
  anchorTitle,
  currentItemId,
  disputeTarget,
  earliestNextReview,
  isRecorded,
  isReview,
  isShownItem,
  itemTurns,
  judgedTurn,
  orderedItems,
  progressAt,
  progressRatio,
  resultRows,
  retryableTurnId,
  selfGradeStep,
  sessionLectureSummary,
  showComposer,
  stepMeta,
  stopCounts,
} from "./lib/checkView";
import { useFollowScroll } from "./lib/useFollowScroll";
import {
  AnswerFields,
  AnswerFooter,
  CheckItemCard,
  CheckResult,
  LectureCompleteAction,
  PlacementResult,
  ReviewNextAction,
  SelfGradeFooter,
  StepFooter,
  SynthesisDoneAction,
  useAnswerForm,
} from "./ui";
import { synthesisRows } from "./lib/synthesis";
import { useLocale, useT } from "@/src/shared/i18n";
import { DIAGNOSTIC_TITLES, REVIEW_STATUS_LABELS, SESSION_STATUS_LABELS, messages } from "./i18n";

const NO_SECTIONS: LectureSection[] = [];

function StatusLine({ children }: { children: ReactNode }) {
  return (
    <p role="status" aria-live="polite" className="flex min-h-touch items-center gap-2 border-b border-line text-sm text-fg-secondary">
      {children}
    </p>
  );
}

/** 진행 줄 "3/12 · 복습" + 진행 막대. */
function StepMeta({ text, ratio }: { text: string; ratio: number }) {
  const t = useT(messages);
  const percent = Math.round(ratio * 100);
  return (
    <div className="pt-4">
      <p className="text-sm text-fg-secondary tabular-nums">{text}</p>
      <div
        role="progressbar"
        aria-label={t.progress}
        aria-valuenow={percent}
        aria-valuemin={0}
        aria-valuemax={100}
        className="mt-2 h-1 w-full overflow-hidden rounded-full bg-canvas-hover"
      >
        <div className="h-full bg-accent-solid" style={{ width: `${percent}%` }} />
      </div>
    </div>
  );
}

/**
 * 복습·확인 세션 화면(T-steps) — 제목·←·페이지 동작은 틀이 그린다. 진행 줄 → 문항(질문이 곧 제목) → 답 입력 + 확신도 →
 * 보낸 뒤 튜터 답·놓친 기준·모범답. 아래 Footer 는 고정: 보내기 전 [그냥 알려줘] [보내기], 문항을 끝낸 뒤 [다음 문항]/[결과 보기].
 * 끝난 세션은 결과 요약(끝 화면)이 맨 위, 문항 기록은 그 아래. 그만두기는 페이지 동작 ⋯ 의 위험 항목 + 확인 창.
 * 채점·문항 생성 중(pending)에만 폴링한다. 복습은 여러 강의의 문항이라 강의 링크를 두지 않는다.
 */
export function CheckView({ checkId }: { checkId: string }) {
  const locale = useLocale();
  const t = useT(messages);
  const check = useCheckSession(checkId);
  const d = check.data;
  const lectureId = d?.session.lectureId ?? "";
  const lecture = useLecture(lectureId);
  const retry = useCheckRetry(checkId);
  const abandon = useCheckAbandon(checkId);
  const resume = useCheckResume(checkId);
  const dispute = useCheckDispute(checkId);
  const confirm = useConfirm();
  const toast = useToast();
  const endRef = useRef<HTMLDivElement>(null);
  const resultRef = useRef<HTMLDivElement>(null);
  const topRef = useRef<HTMLDivElement>(null);
  const sections = lecture.data?.content?.sections ?? NO_SECTIONS;
  const status = d?.session.status;
  const turnCount = d?.turns.length ?? 0;
  const review = d ? isReview(d) : false;
  const statusLabels = (review ? REVIEW_STATUS_LABELS : SESSION_STATUS_LABELS)[locale];

  const anchorLink = useCallback(
    (anchor: string) =>
      lectureId ? { href: lectureAnchorHref(lectureId, anchor), label: anchorTitle(sections, anchor) || t.viewInLecture } : null,
    [lectureId, sections, t],
  );
  const rows = useMemo(() => (d ? resultRows(d) : []), [d]);
  const synthesis = useMemo(() => (d ? synthesisRows(d) : []), [d]);
  const acknowledged = useMemo(() => new Set(d?.misconceptionTurnIds ?? []), [d?.misconceptionTurnIds]);

  const currentId = d ? currentItemId(d) : null;
  const lastTurn = d?.turns[d.turns.length - 1];
  const lastTurnKey = lastTurn ? `${lastTurn.id}:${lastTurn.status}` : "";
  const markAct = useFollowScroll({ status, turnCount, lastTurnKey, currentId, endRef, resultRef });

  // 지금 문항이 바뀌었는데 방금 끝낸 문항이면(서버는 바로 다음 문항으로 넘긴다) 그 문항의 튜터 답·모범답을 붙잡고
  // [다음 문항]/[결과 보기]를 누를 때까지 보인다. 렌더 중에 파생 상태를 갱신한다(effect 를 거치면 다음 문항이 한 번 비친다).
  const [heldId, setHeldId] = useState<string | null>(null);
  const [lastCurrent, setLastCurrent] = useState<string | null>(null);
  if (d && currentId !== lastCurrent) {
    setLastCurrent(currentId);
    if (lastCurrent && !isRecorded(d) && d.items.some((item) => item.id === lastCurrent && item.done)) { setHeldId(lastCurrent); }
  }
  const held = d && heldId ? d.items.find((item) => item.id === heldId) ?? null : null;

  // 문항을 넘기면 그 머리로 — 처음 그릴 때는 움직이지 않는다.
  const focusKey = held?.id ?? "";
  const mounted = useRef(false);
  useEffect(() => {
    if (!mounted.current) {
      mounted.current = true;
      return;
    }
    (topRef.current ?? resultRef.current)?.scrollIntoView?.({ block: "start" });
  }, [focusKey]);

  const kind = d?.session.kind;
  const diagnosticTitle = kind && kind in DIAGNOSTIC_TITLES[locale] ? DIAGNOSTIC_TITLES[locale][kind as keyof (typeof DIAGNOSTIC_TITLES)["ko"]] : "";
  const title = review ? t.reviewTitle : diagnosticTitle || t.checkTitle;
  const open = status === "active" || status === "preparing";
  useScreenTitle(title);

  // 그만두기(되돌릴 수 없음): 답한 데까지로 세션을 닫는다 → 실행 전에 앱 안 확인 창(기본 포커스 취소).
  const answeredCurrent = d && currentId ? itemTurns(d.turns, currentId).some((turn) => turn.role === "learner") : false;
  async function stop() {
    if (!d) { return; }
    const { answered, remaining } = stopCounts(d, answeredCurrent);
    const ok = await confirm({
      title: t.stopConfirmTitle(title),
      description: review ? t.stopConfirmReview(answered, remaining) : t.stopConfirmOther(answered, remaining),
      confirmLabel: t.menuStop,
      tone: "danger",
    });
    if (ok) { abandon.mutate(); }
  }
  // 배치 확인 건너뛰기: 기록만 하는 진단이라 확인 창 없이 닫고 되돌리기 알림을 둔다.
  function skipPlacement() {
    abandon.mutate(undefined, {
      onSuccess: () => {
        toast.undo(t.skippedToast, () => {
          resume.mutateAsync().catch(() => toast.show(t.undoFailed, { tone: "danger" }));
        });
      },
    });
  }

  const actions: PageAction[] = [];
  if (lectureId) { actions.push({ id: "lecture", label: t.viewInLecture, href: lectureHref(lectureId) }); }
  if (open && kind !== "placement" && !abandon.isPending) {
    actions.push({ id: "stop", label: t.menuStop, danger: true, onSelect: () => void stop() });
  }
  usePageActions(actions.length > 0 ? actions : null);

  return <div className="flex w-full flex-col">{renderBody()}</div>;

  function renderBody(): ReactNode {
    if (!d && check.error) {
      return <ErrorState title={describeCheckError(check.error)} onRetry={() => check.refetch()} />;
    }
    if (!d) {
      return (
        <div aria-busy="true" className="flex flex-col gap-4 pt-4">
          <Skeleton className="w-24" />
          <Skeleton lines={3} />
          <Skeleton lines={3} className="h-24" />
        </div>
      );
    }
    if (d.session.status === "preparing") {
      return (
        <div aria-busy="true">
          <StatusLine>
            <ProgressCircle size="24" tone="inherit" aria-hidden />
            {statusLabels.preparing}
          </StatusLine>
          <div className="mt-6 flex flex-col gap-4" aria-hidden="true">
            <Skeleton lines={3} />
            <Skeleton lines={3} className="h-24" />
          </div>
        </div>
      );
    }
    if (d.session.status === "failed") {
      return (
        <div className="pt-4">
          <ErrorState
            title={statusLabels.failed}
            onRetry={() => retry.mutate()}
          />
          {retry.error && <p role="alert" className="mt-2 text-sm text-danger">{describeCheckError(retry.error)}</p>}
        </div>
      );
    }
    if (d.items.length === 0) {
      return <EmptyState title={t.noItems} />;
    }
    if (held) { return renderHeld(d, held); }
    if (d.session.status === "active") { return renderStep(d); }
    return renderFinished(d);
  }

  function itemCard(detail: CheckDetail, item: CheckItem, index: number, total: number, showIndex: boolean): ReactNode {
    const retryId = retryableTurnId(detail);
    const turns = itemTurns(detail.turns, item.id);
    const target = disputeTarget(detail, item, Date.now());
    const link = item.anchor ? anchorLink(item.anchor) : null;
    return (
      <CheckItemCard
        key={item.id}
        index={index}
        total={total}
        item={item}
        turns={turns}
        missing={judgedTurn(detail.turns, item.id)?.grade?.missing ?? []}
        anchorHref={link?.href ?? null}
        anchorLabel={link?.label ?? ""}
        retryTurnId={retryId}
        onRetry={() => retry.mutate()}
        retrying={retry.isPending}
        onDispute={target ? () => dispute.mutate({ turnId: target.id }) : undefined}
        disputing={dispute.isPending && dispute.variables?.turnId === target?.id}
        showLectureSummary={review}
        showIndex={showIndex}
      />
    );
  }

  /** 방금 끝낸 문항 — 튜터 답·놓친 기준·모범답을 보고 [다음 문항]/[결과 보기]. */
  function renderHeld(detail: CheckDetail, item: CheckItem): ReactNode {
    const progress = progressAt(detail, item.id);
    const last = currentId === null;
    return (
      <>
        <div ref={topRef} className="scroll-mt-28">
          <StepMeta text={stepMeta(progress, title)} ratio={progressRatio(progress.index, progress.total, true)} />
        </div>
        <div className="flex flex-col gap-6 pt-4" aria-busy={detail.pending}>
          {itemCard(detail, item, progress.index, progress.total, false)}
        </div>
        {dispute.error && <p role="alert" className="mt-4 text-sm text-danger">{describeCheckError(dispute.error)}</p>}
        <StepFooter>
          <span className="flex-1" />
          <Button variant="primary" onClick={() => setHeldId(null)} disabled={detail.pending}>
            {last ? t.seeResult : t.nextItem}
          </Button>
        </StepFooter>
      </>
    );
  }

  /** 진행 중 — 지금 문항 하나와 아래 고정 Footer. */
  function renderStep(detail: CheckDetail): ReactNode {
    const item = detail.items.find((candidate) => candidate.id === currentId) ?? null;
    const progress = progressAt(detail, currentId);
    const actionError = retry.error ?? abandon.error ?? dispute.error;
    return (
      <>
        <div ref={topRef} className="scroll-mt-28">
          <StepMeta text={stepMeta(progress, title)} ratio={progressRatio(progress.index, progress.total, false)} />
        </div>
        <div className="flex flex-col gap-6 pt-4" aria-busy={detail.pending}>
          {item && itemCard(detail, item, progress.index, progress.total, false)}
        </div>
        <div ref={endRef} />
        {actionError && (
          <p role="alert" className="mt-4 text-sm text-danger">
            {describeCheckError(actionError)}
          </p>
        )}
        {showComposer(detail) && (
          <div className="contents" onPointerDownCapture={markAct} onKeyDownCapture={markAct}>
            {isRecorded(detail) ? renderRecordedFooter(detail) : <AnswerStep key={detail.current?.itemId} detail={detail} />}
          </div>
        )}
      </>
    );
  }

  /** 예측·배치 문항 — 고르거나 쓰면 바로 기록되고 다음 문항으로 간다. 배치 확인은 Footer 에서 건너뛸 수 있다. */
  function renderRecordedFooter(detail: CheckDetail): ReactNode {
    const current = currentDiagnosticItem(detail);
    if (!current) { return null; }
    return (
      <StepFooter>
        <div className="flex min-w-0 flex-1 flex-col gap-2">
          <DiagnosticAnswer key={current.id} detail={detail} item={current} />
          {detail.session.kind === "placement" && (
            <div>
              <Button variant="ghost" onClick={skipPlacement} loading={abandon.isPending}>
                {t.skip}
              </Button>
            </div>
          )}
        </div>
      </StepFooter>
    );
  }

  /** 끝났거나 그만둔 세션 — 결과 요약(끝 화면)이 맨 위, 문항 기록은 그 아래. */
  function renderFinished(detail: CheckDetail): ReactNode {
    const items = orderedItems(detail);
    const statusLabel = detail.session.status === "finished" ? "" : statusLabels[detail.session.status];
    const shown = items.filter((item) => isShownItem(item, null, itemTurns(detail.turns, item.id).length));
    const count = detail.items.length;
    return (
      <>
        {statusLabel && <StatusLine>{statusLabel}</StatusLine>}
        {detail.session.status === "finished" && detail.placement && <PlacementResult ref={resultRef} placement={detail.placement} />}
        {detail.session.status === "finished" && !isRecorded(detail) && (
          <CheckResult
            ref={resultRef}
            checkId={checkId}
            headline={review ? t.finishedReview(count) : t.finishedItems(count)}
            nextReviewAt={earliestNextReview(detail.session.result?.items)}
            rows={rows}
            acknowledgedTurnIds={acknowledged}
            anchorLink={anchorLink}
            lectureSummary={review ? undefined : sessionLectureSummary(detail)}
            synthesis={synthesis}
          >
            {kind === "synthesis" && <SynthesisDoneAction />}
            {review && <ReviewNextAction />}
            {!review && kind !== "synthesis" && lectureId && (
              <LectureCompleteAction lectureId={lectureId} lectureCompleted={Boolean(lecture.data?.progress?.completedAt)} />
            )}
          </CheckResult>
        )}
        <div className="flex flex-col gap-6 pt-4" aria-busy={detail.pending}>
          {shown.map((item) => itemCard(detail, item, items.indexOf(item) + 1, items.length, shown.length > 1))}
        </div>
        <div ref={endRef} />
        {(retry.error ?? abandon.error ?? dispute.error) && (
          <p role="alert" className="mt-4 text-sm text-danger">
            {describeCheckError(retry.error ?? abandon.error ?? dispute.error)}
          </p>
        )}
      </>
    );
  }
}

/** 지금 문항의 답 — 자기 채점에서 답을 보냈으면 채점 Footer, 아니면 입력란(본문) + 확신도 + 보내기 Footer. */
function AnswerStep({ detail }: { detail: CheckDetail }) {
  const form = useAnswerForm(detail);
  if (selfGradeStep(detail) === "grade") { return <SelfGradeFooter key={detail.current?.itemId} id={detail.session.id} />; }
  return (
    <>
      <div className="pt-6">
        <AnswerFields detail={detail} form={form} />
      </div>
      <AnswerFooter detail={detail} form={form} />
    </>
  );
}
