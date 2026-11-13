
import { useCallback, useEffect, useMemo, useRef, type ReactNode } from "react";
import { EmptyState, ErrorState, Skeleton, Spinner, type MenuItem } from "@/src/shared/ui/kit";
import {
  describeCheckError,
  lectureHref,
  useCheckAbandon,
  useCheckDispute,
  useCheckRetry,
  useCheckSession,
  useLecture,
} from "@/src/entities/punisher";
import { describePunisherError, type CheckDetail, type LectureSection } from "@/src/shared/api/punisher";
import { LectureLayout, LectureTopBar } from "../lecture-viewer";
import {
  anchorTitle,
  currentItemId,
  disputeTarget,
  itemTurns,
  judgedTurn,
  orderedItems,
  progressOf,
  resultRows,
  retryableTurnId,
  showComposer,
} from "./lib/checkView";
import { CheckComposer, CheckItemCard, CheckResult } from "./ui";
import { SESSION_STATUS_LABELS } from "./ui/constants";

const NO_SECTIONS: LectureSection[] = [];

function StatusLine({ children }: { children: ReactNode }) {
  return (
    <p role="status" aria-live="polite" className="flex min-h-touch items-center gap-2 border-b border-line text-sm text-fg-secondary">
      {children}
    </p>
  );
}

/**
 * 숙달 확인 화면 — 강의와 같은 상단 바(← 들어온 곳 · 제목 · 문항 n/N · ⋯ 포기)와 가독 폭 본문.
 * 문항마다 물음·자료 → 문답 → 판정, 하단 고정 입력창. 채점·문항 생성 중(pending)에만 폴링한다.
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

  const anchorLink = useCallback(
    (anchor: string) =>
      lectureId ? { href: `${lectureHref(lectureId)}#pa-${encodeURIComponent(anchor)}`, label: anchorTitle(sections, anchor) || "강의에서 보기" } : null,
    [lectureId, sections],
  );
  const rows = useMemo(() => (d ? resultRows(d) : []), [d]);

  // 새 턴이 오면 마지막으로, 끝나면 결과 요약으로 데려간다.
  useEffect(() => {
    if (status === "finished") {
      resultRef.current?.scrollIntoView({ behavior: "smooth", block: "start" });
      return;
    }
    if (status === "active" && turnCount > 0) { endRef.current?.scrollIntoView({ behavior: "smooth", block: "nearest" }); }
  }, [status, turnCount]);

  const title = lecture.data?.content?.title || lecture.data?.title || "확인하기";
  const progress = d ? progressOf(d) : null;
  const barTitle = progress && progress.total > 0 ? `${title} · 문항 ${progress.index}/${progress.total}` : title;
  const open = status === "active" || status === "preparing";
  const menuItems: MenuItem[] = [
    { key: "abandon", label: "포기", danger: true, disabled: !open || abandon.isPending, onSelect: () => abandon.mutate() },
  ];

  return (
    <div className="flex w-full flex-col">
      <LectureTopBar title={barTitle} backHref={lectureId ? lectureHref(lectureId) : undefined} menuItems={d ? menuItems : undefined} />
      <h1 className="sr-only">{barTitle}</h1>
      <LectureLayout sections={NO_SECTIONS} totalMinutes={0} activeAnchor="">
        {renderBody()}
      </LectureLayout>
    </div>
  );

  function renderBody(): ReactNode {
    if (!d && check.error) {
      return <ErrorState message={describePunisherError(check.error)} onRetry={() => check.refetch()} retrying={check.isRefetching} />;
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
            {SESSION_STATUS_LABELS.preparing}
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
            message={d.session.error || SESSION_STATUS_LABELS.failed}
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
    const currentId = currentItemId(detail);
    const retryId = retryableTurnId(detail);
    const now = Date.now();
    const actionError = retry.error ?? abandon.error ?? dispute.error;
    const statusLabel = SESSION_STATUS_LABELS[detail.session.status];
    return (
      <>
        {statusLabel && <StatusLine>{statusLabel}</StatusLine>}
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
        {detail.session.status === "finished" && (
          <CheckResult
            ref={resultRef}
            lectureId={lectureId}
            rows={rows}
            anchorLink={anchorLink}
            lectureCompleted={Boolean(lecture.data?.progress?.completedAt)}
          />
        )}
        {showComposer(detail) && <CheckComposer detail={detail} />}
      </>
    );
  }
}
