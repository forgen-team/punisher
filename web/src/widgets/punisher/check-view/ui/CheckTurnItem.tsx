
import { Badge, Button, Spinner } from "@/src/shared/ui/kit";
import { SELF_VERDICT_LABELS, VERDICT_LABELS, type SelfVerdict } from "@/src/entities/punisher";
import type { CheckTurn } from "@/src/shared/api/punisher";
import { Markdown } from "@/src/shared/ui/punisher/Markdown";
import { DISPUTE_RESULT_LABELS, PENDING_LABELS, VERDICT_TONE } from "./constants";

interface CheckTurnItemProps {
  turn: CheckTurn;
  /** 이 턴이 '다시 시도'할 실패 턴이면 */
  onRetry?: () => void;
  retrying?: boolean;
}

/** 글 없는 학습자 턴의 표시 — 이의 제기, 자기 채점(고른 판정). */
function learnerActionText(turn: CheckTurn): string {
  if (turn.move === "dispute") { return "내 답이 맞아요"; }
  if (turn.move === "self_grade" && turn.verdict) { return SELF_VERDICT_LABELS[turn.verdict as SelfVerdict] ?? ""; }
  return "";
}

/** 학습자 답(오른쪽) / 튜터 응답(왼쪽, 판정 배지) — 질문 해설의 후속 문답과 같은 대기·실패 처리. */
export function CheckTurnItem({ turn, onRetry, retrying }: CheckTurnItemProps) {
  if (turn.role === "learner") {
    const text = turn.text || learnerActionText(turn);
    return (
      <li className="flex justify-end">
        <p className="max-w-[85%] whitespace-pre-wrap break-words rounded-lg bg-accent-subtle px-3 py-2 text-sm text-fg">
          <span className="sr-only">내 답: </span>
          {text}
        </p>
      </li>
    );
  }
  if (turn.status === "pending") {
    return (
      <li className="flex items-center gap-2 text-sm text-fg-secondary">
        <Spinner size="sm" label={null} />
        {turn.move === "dispute" ? PENDING_LABELS.dispute : PENDING_LABELS.turn}
      </li>
    );
  }
  if (turn.status === "failed") {
    return (
      <li className="flex flex-wrap items-center gap-2 text-sm text-warning">
        답을 만들지 못했어요.
        {onRetry && (
          <Button size="sm" variant="secondary" loading={retrying} onClick={onRetry}>
            다시 시도
          </Button>
        )}
      </li>
    );
  }
  const accepted = turn.move === "dispute" && turn.grade?.accepted !== undefined ? turn.grade.accepted : undefined;
  return (
    <li className="flex flex-col gap-1.5 rounded-lg bg-canvas-subtle px-3 py-2 text-sm">
      <span className="sr-only">튜터: </span>
      {(turn.verdict || accepted !== undefined) && (
        <span className="flex flex-wrap items-center gap-1.5">
          {turn.verdict && <Badge tone={VERDICT_TONE[turn.verdict]}>{VERDICT_LABELS[turn.verdict]}</Badge>}
          {accepted !== undefined && (
            <Badge tone={accepted ? "success" : "neutral"}>
              {accepted ? DISPUTE_RESULT_LABELS.accepted : DISPUTE_RESULT_LABELS.rejected}
            </Badge>
          )}
        </span>
      )}
      {turn.text && <Markdown>{turn.text}</Markdown>}
    </li>
  );
}
