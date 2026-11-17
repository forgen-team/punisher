
import { Badge, Button, Spinner } from "@/src/shared/ui/kit";
import { SELF_VERDICT_LABELS_BY_LOCALE, VERDICT_LABELS_BY_LOCALE, type SelfVerdict } from "@/src/entities/punisher";
import type { CheckTurn } from "@/src/shared/api/punisher";
import { Markdown } from "@/src/shared/ui/punisher/Markdown";
import { isRevealNotice, turnBody } from "../lib/checkView";
import { translate, useLocale, useT, type Locale } from "@/src/shared/i18n";
import { DISPUTE_RESULT_LABELS, PENDING_LABELS, messages } from "../i18n";
import { VERDICT_TONE } from "./constants";

interface CheckTurnItemProps {
  turn: CheckTurn;
  /** 이 턴이 '다시 시도'할 실패 턴이면 */
  onRetry?: () => void;
  retrying?: boolean;
}

/** 글 없는 학습자 턴의 표시 — 이의 제기, 자기 채점(고른 판정). */
function learnerActionText(turn: CheckTurn, locale: Locale): string {
  if (turn.move === "dispute") { return translate(messages, locale).dispute; }
  if (turn.move === "self_grade" && turn.verdict) { return SELF_VERDICT_LABELS_BY_LOCALE[locale][turn.verdict as SelfVerdict] ?? ""; }
  return "";
}

/** 학습자 답(오른쪽) / 튜터 응답(왼쪽, 판정 배지) — 질문 해설의 후속 문답과 같은 대기·실패 처리. */
export function CheckTurnItem({ turn, onRetry, retrying }: CheckTurnItemProps) {
  const locale = useLocale();
  const t = useT(messages);
  if (turn.role === "learner") {
    const text = turn.text || learnerActionText(turn, locale);
    return (
      <li className="flex justify-end">
        <p className="max-w-[85%] whitespace-pre-wrap break-words rounded-lg bg-accent-subtle px-3 py-2 text-sm text-fg">
          <span className="sr-only">{t.myAnswer}</span>
          {text}
        </p>
      </li>
    );
  }
  if (turn.status === "pending") {
    return (
      <li className="flex items-center gap-2 text-sm text-fg-secondary">
        <Spinner size="sm" label={null} />
        {turn.move === "dispute" ? PENDING_LABELS[locale].dispute : PENDING_LABELS[locale].turn}
      </li>
    );
  }
  if (turn.status === "failed") {
    return (
      <li className="flex flex-wrap items-center gap-2 text-sm text-warning">
        {t.turnFailed}
        {onRetry && (
          <Button size="sm" variant="secondary" loading={retrying} onClick={onRetry}>
            {t.retry}
          </Button>
        )}
      </li>
    );
  }
  if (isRevealNotice(turn)) {
    return <li className="text-xs text-fg-muted">{t.revealedNotice}</li>;
  }
  const accepted = turn.move === "dispute" && turn.grade?.accepted !== undefined ? turn.grade.accepted : undefined;
  const body = turnBody(turn);
  return (
    <li className="flex flex-col gap-1.5 rounded-lg bg-canvas-muted px-3 py-2 text-sm">
      <span className="sr-only">{t.tutor}</span>
      {(turn.verdict || accepted !== undefined) && (
        <span className="flex flex-wrap items-center gap-1.5">
          {turn.verdict && <Badge tone={VERDICT_TONE[turn.verdict]}>{VERDICT_LABELS_BY_LOCALE[locale][turn.verdict]}</Badge>}
          {accepted !== undefined && (
            <Badge tone={accepted ? "success" : "neutral"}>
              {accepted ? DISPUTE_RESULT_LABELS[locale].accepted : DISPUTE_RESULT_LABELS[locale].rejected}
            </Badge>
          )}
          {accepted && turn.grade?.conflict && <Badge tone="warning">{DISPUTE_RESULT_LABELS[locale].conflict}</Badge>}
        </span>
      )}
      {body && <Markdown>{body}</Markdown>}
    </li>
  );
}
