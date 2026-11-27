import type { ReactNode } from "react";
import { Link } from "@/src/shared/lib/router";
import { Button } from "@forgen-team/ui";
import { describeCheckError } from "@/src/entities/punisher";
import { useLaunchCheck, useLaunchReview, useLaunchSynthesis } from "@/src/features/punisher/check-start";
import { useT } from "@/src/shared/i18n";
import { messages } from "../i18n";
import type { ReviewBundleState, ReviewCounts, ReviewTarget } from "../lib/reviewBundle";
import { BoxPad, Bundle } from "./TodayBox";

export const CARDS_PATH = "/cards";

/** [복습 시작]·[그래도 복습]·[오늘 더 하기]·[이어서 복습] 이 여는 세션 — 지금 복습 시작 훅을 그대로 쓴다. */
function useStartTarget(target: ReviewTarget) {
  const review = useLaunchReview();
  const check = useLaunchCheck(target.kind === "check" ? target.lectureId : "");
  const synthesis = useLaunchSynthesis();
  const launcher = target.kind === "check" ? check : target.kind === "synthesis" ? synthesis : review;
  return {
    launch: () => { if (!launcher.isPending) { launcher.launch(); } },
    isPending: launcher.isPending,
    error: launcher.error,
  };
}

/** "복습 12 · 확인 대기 3 · 카드 8 · 종합 1" — 0 인 항목은 뺀다. 복습만 굵게. */
function CountLine({ counts }: { counts: ReviewCounts }) {
  const t = useT(messages);
  const parts: ReactNode[] = [];
  if (counts.review > 0) { parts.push(<strong key="review" className="font-semibold text-fg">{t.countReview(counts.review)}</strong>); }
  if (counts.confirm > 0) { parts.push(<span key="confirm">{t.countConfirm(counts.confirm)}</span>); }
  if (counts.cards > 0) { parts.push(<span key="cards">{t.countCards(counts.cards)}</span>); }
  if (counts.synthesis > 0) { parts.push(<span key="synthesis">{t.countSynthesis(counts.synthesis)}</span>); }
  return (
    <p className="text-sm tabular-nums text-fg">
      {parts.map((part, index) => (
        <span key={index}>
          {index > 0 && " · "}
          {part}
        </span>
      ))}
    </p>
  );
}

function CardsLink({ cards }: { cards: number }) {
  const t = useT(messages);
  if (cards <= 0) { return null; }
  return (
    <Link href={CARDS_PATH} className="text-sm text-accent-text underline-offset-2 hover:underline">
      {t.reviewCards}
    </Link>
  );
}

/** 처리할 것 › 복습 묶음 — 상태 5종(숫자 줄·다 함·쉬는 날·없음·중간에 나감). */
export function ReviewBundle({ state }: { state: ReviewBundleState }) {
  const t = useT(messages);
  // 다 함·중간에 나감은 오늘 복습 세션을 다시 받는다(끝났으면 새로, 열려 있으면 그것).
  const target: ReviewTarget = state.kind === "due" || state.kind === "rest" ? state.target : { kind: "review" };
  const start = useStartTarget(target);
  const { counts } = state;

  let body: ReactNode;
  if (state.kind === "none") {
    body = <p className="text-sm text-fg-secondary">{t.reviewNone}</p>;
  } else if (state.kind === "partial") {
    body = (
      <>
        <p className="text-sm text-fg">
          <strong className="font-semibold">{t.resumeReview}</strong>
          <span className="tabular-nums text-fg-secondary"> · {t.reviewLeft(state.left)}</span>
        </p>
        <Actions>
          <Button variant="primary" size="sm" onClick={start.launch} loading={start.isPending}>{t.resumeReview}</Button>
          <CardsLink cards={counts.cards} />
        </Actions>
      </>
    );
  } else if (state.kind === "done") {
    body = (
      <>
        <p className="text-sm text-fg">
          <strong className="font-semibold">{t.reviewDone}</strong>
          <span className="tabular-nums text-fg-secondary"> · {state.done}/{state.total}</span>
        </p>
        <Actions>
          <Button variant="secondary" size="sm" onClick={start.launch} loading={start.isPending}>{t.doMore}</Button>
          <CardsLink cards={counts.cards} />
        </Actions>
      </>
    );
  } else if (state.kind === "rest") {
    body = (
      <>
        <p className="text-sm text-fg">{t.restDay}</p>
        <Actions>
          {state.target.kind !== "none" && (
            <Button variant="secondary" size="sm" onClick={start.launch} loading={start.isPending}>{t.reviewAnyway}</Button>
          )}
          <CardsLink cards={counts.cards} />
        </Actions>
      </>
    );
  } else {
    body = (
      <>
        <CountLine counts={counts} />
        {counts.minutes > 0 && <p className="mt-1 text-xs tabular-nums text-fg-muted">{t.aboutMinutes(counts.minutes)}</p>}
        <Actions>
          {state.target.kind !== "none" && (
            <Button variant="primary" size="sm" onClick={start.launch} loading={start.isPending}>{t.startReview}</Button>
          )}
          <CardsLink cards={counts.cards} />
        </Actions>
      </>
    );
  }

  return (
    <Bundle name={t.review}>
      <BoxPad>
        {body}
        {start.error ? (
          <p role="alert" className="mt-2 text-sm text-danger">
            {describeCheckError(start.error)}
          </p>
        ) : null}
      </BoxPad>
    </Bundle>
  );
}

function Actions({ children }: { children: ReactNode }) {
  return <div className="mt-3 flex flex-wrap items-center gap-3">{children}</div>;
}
