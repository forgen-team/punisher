import { useState, type ReactNode } from "react";
import { Button, Textarea } from "@/src/shared/ui/kit";
import { describePunisherError, EXERCISE_MAX_ANSWER_CHARS } from "@/src/shared/api/punisher";
import { blockContentHash } from "../lib/contentHash";
import { findAttempt, useExerciseContext } from "../model/ExerciseContext";
import { AttemptFeedback } from "./AttemptFeedback";
import { EXERCISE_LABELS } from "./constants";

interface ExercisePanelProps {
  anchor: string;
  problem: string;
  solution: string;
  /** 풀이(본문과 같은 Markdown) — 답하거나 건너뛰면 펼친다 */
  solutionView: ReactNode;
  /** 문맥이 없을 때(미리보기) 그릴 기존 접힌 풀이 */
  fallback: ReactNode;
}

/** '직접 해 보기' — 한 줄이라도 쓰면 풀이가 열리고 튜터 평이 뒤따른다(건너뛰기 가능). */
export function ExercisePanel({ anchor, problem, solution, solutionView, fallback }: ExercisePanelProps) {
  const ctx = useExerciseContext();
  const [text, setText] = useState("");
  if (!ctx || !anchor) { return <>{fallback}</>; }
  const hash = blockContentHash(problem, solution);
  const attempt = findAttempt(ctx.attempts, anchor, hash);
  const busy = ctx.submitting === anchor;
  const error = ctx.submitErrorAnchor === anchor ? ctx.submitError : null;

  if (attempt) {
    return (
      <div className="mt-2">
        {attempt.answer && (
          <p className="whitespace-pre-wrap break-keep rounded-sm bg-canvas-subtle px-2 py-1.5 text-sm text-fg-secondary">
            <span className="sr-only">{EXERCISE_LABELS.myAnswer}: </span>
            {attempt.answer}
          </p>
        )}
        <div className="mt-2 border-l-2 border-line-strong pl-3">{solutionView}</div>
        <AttemptFeedback attempt={attempt} />
      </div>
    );
  }
  const send = (skip: boolean) => {
    void ctx.submit({ anchor, contentHash: hash, ...(skip ? { skip: true } : { answer: text }) }).catch(() => undefined);
  };
  return (
    <div className="mt-2 flex flex-col gap-2">
      <Textarea
        aria-label={EXERCISE_LABELS.answerPlaceholder}
        placeholder={EXERCISE_LABELS.answerPlaceholder}
        rows={2}
        maxLength={EXERCISE_MAX_ANSWER_CHARS}
        value={text}
        onChange={(event) => setText(event.target.value)}
        className="resize-y"
      />
      <div className="flex flex-wrap items-center gap-2">
        <Button size="sm" loading={busy} disabled={!text.trim() || busy} onClick={() => send(false)}>
          {EXERCISE_LABELS.submit}
        </Button>
        <Button size="sm" variant="ghost" disabled={busy} onClick={() => send(true)}>
          {EXERCISE_LABELS.skip}
        </Button>
      </div>
      {error != null && <p role="alert" className="text-sm text-danger">{describePunisherError(error)}</p>}
    </div>
  );
}
