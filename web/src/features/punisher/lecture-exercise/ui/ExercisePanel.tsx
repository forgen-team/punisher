import { useState, type ReactNode } from "react";
import { Button, TextField } from "@forgen-team/ui";
import { describePunisherError, EXERCISE_MAX_ANSWER_CHARS } from "@/src/shared/api/punisher";
import { blockContentHash } from "../lib/contentHash";
import { findAttempt, useExerciseContext } from "../model/ExerciseContext";
import { AttemptFeedback } from "./AttemptFeedback";
import { useT } from "@/src/shared/i18n";
import { messages } from "../i18n";

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
  const t = useT(messages);
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
            <span className="sr-only">{t.myAnswer}: </span>
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
      <TextField
        multiline
        placeholder={t.answerPlaceholder}
        value={text}
        onValueChange={({ value }) => setText(value)}
        textareaProps={{ "aria-label": t.answerPlaceholder, rows: 2, maxLength: EXERCISE_MAX_ANSWER_CHARS, className: "resize-y" }}
      />
      <div className="flex flex-wrap items-center gap-2">
        <Button variant="primary" size="sm" loading={busy} disabled={!text.trim() || busy} onClick={() => send(false)}>
          {t.submit}
        </Button>
        <Button size="sm" variant="ghost" disabled={busy} onClick={() => send(true)}>
          {t.skip}
        </Button>
      </div>
      {error != null && <p role="alert" className="text-sm text-danger">{describePunisherError(error)}</p>}
    </div>
  );
}
