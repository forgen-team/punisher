
import { useCallback, useMemo, useRef, type ReactNode } from "react";
import { EmptyState, ErrorState, Skeleton, Spinner, type MenuItem } from "@/src/shared/ui/kit";
import {
  describeCheckError,
  lectureHref,
  useCheckAbandon,
  useCheckDispute,
  useCheckRetry,
  useCheckSession,
  useLecture,
  useReviewSummary,
} from "@/src/entities/punisher";
import type { CheckDetail, LectureSection } from "@/src/shared/api/punisher";
import { DiagnosticAnswer, currentDiagnosticItem } from "@/src/features/punisher/pretest";
import { LectureLayout, LectureTopBar } from "../lecture-viewer";
import {
  anchorTitle,
  currentItemId,
  disputeTarget,
  isRecorded,
  isReview,
  itemTurns,
  judgedTurn,
  orderedItems,
  progressLabel,
  resultRows,
  retryableTurnId,
  sessionLectureSummary,
  showComposer,
  topBarTitle,
} from "./lib/checkView";
import { useFollowScroll } from "./lib/useFollowScroll";
import { CheckComposer, CheckItemCard, CheckResult, LectureCompleteAction, PlacementResult, ReviewMoreAction, chapterMapHref } from "./ui";
import { DIAGNOSTIC_TITLES, REVIEW_STATUS_LABELS, SESSION_STATUS_LABELS } from "./ui/constants";

const NO_SECTIONS: LectureSection[] = [];

function StatusLine({ children }: { children: ReactNode }) {
  return (
    <p role="status" aria-live="polite" className="flex min-h-touch items-center gap-2 border-b border-line text-sm text-fg-secondary">
      {children}
    </p>
  );
}

/**
 * 숙달 확인·복습 화면 — 강의와 같은 상단 바(← 들어온 곳 · '제목 · 문항 n/N' · ⋯ 포기)와 가독 폭 본문.
 * 끝난 세션은 결과 요약을 맨 위(상태 줄 아래)에 두고, 문항 기록은 그 아래에 둔다.
 * 문항마다 물음·자료 → 문답 → 판정(자기 채점은 답 → 모범답·기준 → 맞음/부분/틀림), 하단 고정 입력창.
 * 채점·문항 생성 중(pending)에만 폴링한다. 복습은 여러 강의의 문항이라 강의 링크를 두지 않는다.
 */
export function CheckView({ checkId }: { checkId: string }) {
  const check = useCheckSession(checkId);
  const d = check.data;
  const lectureId = d?.session.lectureId ?? "";
  const lecture = useLecture(lectureId);
  const retry = useCheckRetry(checkId);
  const abandon = useCheckAbandon(checkId);
  const dispute = useCheckDispute(checkId);
  const endRef = useRef<HTMLDivElement>(null);
  const resultRef = useRef<HTMLElement>(null);
  const sections = lecture.data?.content?.sections ?? NO_SECTIONS;
  const status = d?.session.status;
  const turnCount = d?.turns.length ?? 0;
  const review = d ? isReview(d) : false;
  const summary = useReviewSummary(review && status === "finished");
  const statusLabels = review ? REVIEW_STATUS_LABELS : SESSION_STATUS_LABELS;

  const anchorLink = useCallback(
    (anchor: string) =>
      lectureId ? { href: `${lectureHref(lectureId)}#pa-${encodeURIComponent(anchor)}`, label: anchorTitle(sections, anchor) || "강의에서 보기" } : null,
    [lectureId, sections],
  );
  const rows = useMemo(() => (d ? resultRows(d) : []), [d]);
  const acknowledged = useMemo(() => new Set(d?.misconceptionTurnIds ?? []), [d?.misconceptionTurnIds]);

  const currentId = d ? currentItemId(d) : null;
  const markAct = useFollowScroll({ status, turnCount, currentId, endRef, resultRef });

  const kind = d?.session.kind;
  const diagnosticTitle = kind && kind in DIAGNOSTIC_TITLES ? DIAGNOSTIC_TITLES[kind as keyof typeof DIAGNOSTIC_TITLES] : "";
  const title = review ? "복습" : diagnosticTitle || lecture.data?.content?.title || lecture.data?.title || "확인하기";
  const chapterId = d?.placement?.chapterId ?? "";
  const chapterHref = chapterId ? chapterMapHref(chapterId) : undefined;
  const backHref = lectureId ? lectureHref(lectureId) : chapterHref;
  const barTitle = topBarTitle(title, d ? progressLabel(d) : "");
  const open = status === "active" || status === "preparing";
  const menuItems: MenuItem[] = [
    {
      key: "abandon",
      label: kind === "placement" ? "건너뛰기" : "포기",
      danger: kind !== "placement",
      disabled: !open || abandon.isPending,
      onSelect: () => abandon.mutate(),
    },
  ];

  return (
    <div className="flex w-full flex-col">
      <LectureTopBar title={barTitle} backHref={backHref} menuItems={d ? menuItems : undefined} />
      <h1 className="sr-only">{barTitle}</h1>
      <LectureLayout sections={NO_SECTIONS} totalMinutes={0} activeAnchor="">
        {renderBody()}
      </LectureLayout>
    </div>
  );

  function renderBody(): ReactNode {
    if (!d && check.error) {
      return <ErrorState message={describeCheckError(check.error)} onRetry={() => check.refetch()} retrying={check.isRefetching} />;
    }
    if (!d) {
      return (
        <div aria-busy="true" className="flex flex-col gap-4 pt-4">
          <Skeleton className="w-24" />
          <Skeleton lines={3} />
          <Skeleton variant="block" className="h-24" />
        </div>
      );
    }
    if (d.session.status === "preparing") {
      return (
        <div aria-busy="true">
          <StatusLine>
            <Spinner size="sm" label={null} />
            {statusLabels.preparing}
          </StatusLine>
          <div className="mt-6 flex flex-col gap-4" aria-hidden="true">
            <Skeleton lines={3} />
            <Skeleton variant="block" className="h-24" />
          </div>
        </div>
      );
    }
    if (d.session.status === "failed") {
      return (
        <div className="pt-4">
          <ErrorState
            message={d.session.error || statusLabels.failed}
            onRetry={() => retry.mutate()}
            retrying={retry.isPending}
          />
          {retry.error && <p role="alert" className="mt-2 text-sm text-danger">{describeCheckError(retry.error)}</p>}
        </div>
      );
    }
    if (d.items.length === 0) {
      return <EmptyState title="확인할 문항이 없어요" />;
    }
    return renderFlow(d);
  }

  function renderFlow(detail: CheckDetail): ReactNode {
    const items = orderedItems(detail);
    const retryId = retryableTurnId(detail);
    const now = Date.now();
    const actionError = retry.error ?? abandon.error ?? dispute.error;
    const statusLabel = statusLabels[detail.session.status];
    return (
      <>
        {statusLabel && <StatusLine>{statusLabel}</StatusLine>}
        {detail.session.status === "finished" && detail.placement && <PlacementResult ref={resultRef} placement={detail.placement} />}
        {detail.session.status === "finished" && !isRecorded(detail) && (
          <CheckResult
            ref={resultRef}
            checkId={checkId}
            rows={rows}
            acknowledgedTurnIds={acknowledged}
            anchorLink={anchorLink}
            lectureSummary={review ? undefined : sessionLectureSummary(detail)}
          >
            {review && <ReviewMoreAction remaining={summary.data?.due ?? 0} />}
            {!review && lectureId && (
              <LectureCompleteAction lectureId={lectureId} lectureCompleted={Boolean(lecture.data?.progress?.completedAt)} />
            )}
          </CheckResult>
        )}
        <div className="flex flex-col gap-6 pt-4" aria-busy={detail.pending}>
          {items.map((item, index) => {
            const turns = itemTurns(detail.turns, item.id);
            if (!item.done && item.id !== currentId && turns.length === 0) { return null; }
            const target = disputeTarget(detail, item, now);
            const link = item.anchor ? anchorLink(item.anchor) : null;
            return (
              <CheckItemCard
                key={item.id}
                index={index + 1}
                total={items.length}
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
              />
            );
          })}
        </div>
        <div ref={endRef} />
        {actionError && (
          <p role="alert" className="mt-4 text-sm text-danger">
            {describeCheckError(actionError)}
          </p>
        )}
        {showComposer(detail) && (
          <div className="contents" onPointerDownCapture={markAct} onKeyDownCapture={markAct}>
            {renderComposer(detail)}
          </div>
        )}
      </>
    );
  }

  function renderComposer(detail: CheckDetail): ReactNode {
    if (!isRecorded(detail)) { return <CheckComposer detail={detail} />; }
    const current = currentDiagnosticItem(detail);
    if (!current) { return null; }
    return (
      <div className="sticky bottom-[var(--app-bottom-offset)] z-nav -mx-4 mt-6 border-t border-line bg-canvas px-4 py-3 fold:-mx-6 fold:px-6 lg:mx-0 lg:px-0">
        <DiagnosticAnswer key={current.id} detail={detail} item={current} />
      </div>
    );
  }
}
