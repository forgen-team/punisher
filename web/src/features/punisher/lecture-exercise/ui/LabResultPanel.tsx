import { useState } from "react";
import { Button, TextField } from "@forgen-team/ui";
import { describePunisherError, LAB_MAX_OUTPUT_BYTES } from "@/src/shared/api/punisher";
import { blockContentHash } from "../lib/contentHash";
import { findAttempt, useExerciseContext } from "../model/ExerciseContext";
import { AttemptFeedback } from "./AttemptFeedback";
import { useT } from "@/src/shared/i18n";
import { messages } from "../i18n";

/** 실습 결과 붙여 넣기 → 튜터가 읽어 준다. 붙여 넣은 출력은 지울 수 있다. */
export function LabResultPanel({ anchor, text: labText }: { anchor: string; text: string }) {
  const t = useT(messages);
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
          <span className="sr-only">{t.labOutput}: </span>
          {attempt.answer}
        </pre>
        <AttemptFeedback attempt={attempt} />
        <Button size="sm" variant="ghost" className="mt-1" onClick={() => ctx.remove(attempt.id)}>
          {t.labClear}
        </Button>
      </div>
    );
  }
  const tooLong = new TextEncoder().encode(output).length > LAB_MAX_OUTPUT_BYTES;
  return (
    <div className="mt-2 flex flex-col gap-2">
      <TextField
        multiline
        placeholder={t.labPlaceholder}
        value={output}
        onValueChange={({ value }) => setOutput(value)}
        textareaProps={{
          "aria-label": t.labOutput,
          rows: 3,
          className: "resize-y font-mono text-xs placeholder:font-sans placeholder:text-sm",
        }}
      />
      <div className="flex flex-wrap items-center gap-2">
        <Button
          size="sm"
          variant="secondary"
          loading={busy}
          disabled={!output.trim() || busy}
          onClick={() => void ctx.submit({ anchor, contentHash: hash, answer: output }).catch(() => undefined)}
        >
          {t.labSubmit}
        </Button>
        {tooLong && <span className="text-xs text-fg-muted">{t.labTooLong}</span>}
      </div>
      {error != null && <p role="alert" className="text-sm text-danger">{describePunisherError(error)}</p>}
    </div>
  );
}
