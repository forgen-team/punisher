import { useState } from "react";
import { Button, Textarea } from "@/src/shared/ui/kit";
import { describePunisherError, LAB_MAX_OUTPUT_BYTES } from "@/src/shared/api/punisher";
import { blockContentHash } from "../lib/contentHash";
import { findAttempt, useExerciseContext } from "../model/ExerciseContext";
import { AttemptFeedback } from "./AttemptFeedback";
import { EXERCISE_LABELS } from "./constants";

/** 실습 결과 붙여 넣기 → 튜터가 읽어 준다. 붙여 넣은 출력은 지울 수 있다. */
export function LabResultPanel({ anchor, text: labText }: { anchor: string; text: string }) {
  const ctx = useExerciseContext();
  const [output, setOutput] = useState("");
  if (!ctx || !anchor || ctx.labOs === "none") { return null; }
  const hash = blockContentHash(labText, "");
  const attempt = findAttempt(ctx.attempts, anchor, hash);
  const busy = ctx.submitting === anchor;
  const error = ctx.submitErrorAnchor === anchor ? ctx.submitError : null;

  if (attempt) {
    return (
      <div className="mt-2">
        <pre className="max-h-48 overflow-auto whitespace-pre rounded-sm bg-canvas-subtle px-2 py-1.5 text-xs text-fg-secondary">
          <span className="sr-only">{EXERCISE_LABELS.labOutput}: </span>
          {attempt.answer}
        </pre>
        <AttemptFeedback attempt={attempt} />
        <Button size="sm" variant="ghost" className="mt-1" onClick={() => ctx.remove(attempt.id)}>
          {EXERCISE_LABELS.labClear}
        </Button>
      </div>
    );
  }
  const tooLong = new TextEncoder().encode(output).length > LAB_MAX_OUTPUT_BYTES;
  return (
    <div className="mt-2 flex flex-col gap-2">
      <Textarea
        aria-label={EXERCISE_LABELS.labOutput}
        placeholder={EXERCISE_LABELS.labPlaceholder}
        rows={3}
        value={output}
        onChange={(event) => setOutput(event.target.value)}
        className="resize-y font-mono text-xs placeholder:font-sans placeholder:text-sm"
      />
      <div className="flex flex-wrap items-center gap-2">
        <Button
          size="sm"
          variant="secondary"
          loading={busy}
          disabled={!output.trim() || busy}
          onClick={() => void ctx.submit({ anchor, contentHash: hash, answer: output }).catch(() => undefined)}
        >
          {EXERCISE_LABELS.labSubmit}
        </Button>
        {tooLong && <span className="text-xs text-fg-muted">앞 8KB만 읽어요</span>}
      </div>
      {error != null && <p role="alert" className="text-sm text-danger">{describePunisherError(error)}</p>}
    </div>
  );
}
