import { useId, useState, type ReactNode } from "react";
import { Button, Chip, Input, Textarea } from "@/src/shared/ui/kit";
import { describePunisherError, PREDICTION_MAX_CHARS, type ExerciseInput, type PredictionConfidence } from "@/src/shared/api/punisher";
import { blockContentHash } from "../lib/contentHash";
import { parseRecall } from "../lib/recall";
import { findAttempt, useExerciseContext } from "../model/ExerciseContext";
import { useT } from "@/src/shared/i18n";
import { messages, RECALL_GRADES } from "../i18n";

const CONFIDENCES: PredictionConfidence[] = ["sure", "unsure", "unknown"];

/** 저장 실패 한 줄(그 블록에서 보낸 요청일 때만) */
function SubmitError({ anchor }: { anchor: string }) {
  const ctx = useExerciseContext();
  if (!ctx || ctx.submitErrorAnchor !== anchor || ctx.submitError == null) { return null; }
  return <p role="alert" className="text-sm text-danger">{describePunisherError(ctx.submitError)}</p>;
}

function useSend(anchor: string) {
  const ctx = useExerciseContext();
  return {
    busy: ctx?.submitting === anchor,
    send: (input: ExerciseInput) => { void ctx?.submit(input).catch(() => undefined); },
  };
}

interface PausePanelProps {
  anchor: string;
  question: string;
  /** 접어 둔 출력 블록의 원문(content hash 용) */
  outputText: string;
  output: ReactNode;
  /** 문맥이 없을 때(미리보기) 그릴 기존 접힌 출력 */
  fallback: ReactNode;
}

/** `@@pause` — 예측을 한 줄 쓰거나 확신도를 고른 뒤 출력을 열고, 내 예측과 같았는지 한 번 누른다(건너뛰기 가능). */
export function PausePanel({ anchor, question, outputText, output, fallback }: PausePanelProps) {
  const t = useT(messages);
  const ctx = useExerciseContext();
  const { busy, send } = useSend(anchor);
  const [prediction, setPrediction] = useState("");
  const [confidence, setConfidence] = useState<PredictionConfidence | undefined>();
  const [revealed, setRevealed] = useState(false);
  const groupId = useId();
  if (!ctx || !anchor) { return <>{fallback}</>; }
  const hash = blockContentHash(question, outputText);
  const attempt = findAttempt(ctx.attempts, anchor, hash);
  if (attempt) {
    const right = attempt.verdict === "correct";
    return (
      <div className="mt-2 flex flex-col gap-2">
        {attempt.answer && (
          <p className="break-keep rounded-sm bg-canvas-subtle px-2 py-1.5 text-sm text-fg-secondary">
            <span className="font-medium">{t.myPrediction}</span>
            {attempt.confidence && ` · ${t.confidence(attempt.confidence)}`}: {attempt.answer}
          </p>
        )}
        {output}
        {attempt.verdict !== "skipped" && (
          <p role="status" className={right ? "text-sm font-semibold text-success" : "text-sm font-semibold text-warning"}>
            {right ? t.predictionRight : t.predictionWrong}
          </p>
        )}
      </div>
    );
  }
  if (!revealed) {
    return (
      <div className="mt-2 flex flex-col gap-2">
        <Input
          aria-label={t.predictPlaceholder}
          placeholder={t.predictPlaceholder}
          maxLength={PREDICTION_MAX_CHARS}
          value={prediction}
          onChange={(event) => setPrediction(event.target.value)}
        />
        <div role="radiogroup" aria-labelledby={groupId} className="flex flex-wrap items-center gap-2">
          <span id={groupId} className="text-sm text-fg-secondary">{t.confidenceLabel}</span>
          {CONFIDENCES.map((c) => (
            <Chip key={c} role="radio" aria-checked={confidence === c} selected={confidence === c} onClick={() => setConfidence(c)}>
              {t.confidence(c)}
            </Chip>
          ))}
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <Button size="sm" disabled={!prediction.trim() && !confidence} onClick={() => setRevealed(true)}>
            {t.revealOutput}
          </Button>
          <Button size="sm" variant="ghost" disabled={busy} onClick={() => send({ anchor, contentHash: hash, skip: true })}>
            {t.skipPredict}
          </Button>
        </div>
        <SubmitError anchor={anchor} />
      </div>
    );
  }
  const grade = (verdict: "correct" | "incorrect") =>
    send({ anchor, contentHash: hash, verdict, answer: prediction.trim(), ...(confidence ? { confidence } : {}) });
  return (
    <div className="mt-2 flex flex-col gap-2">
      {output}
      <p className="text-sm text-fg-secondary">{t.sameAsPrediction}</p>
      <div className="flex flex-wrap items-center gap-2">
        <Button size="sm" variant="secondary" loading={busy} disabled={busy} onClick={() => grade("correct")}>{t.predictionRight}</Button>
        <Button size="sm" variant="secondary" disabled={busy} onClick={() => grade("incorrect")}>{t.predictionWrong}</Button>
      </div>
      <SubmitError anchor={anchor} />
    </div>
  );
}

interface RecallPanelProps {
  anchor: string;
  text: string;
  /** 질문·답을 본문과 같은 Markdown 으로 그린다 */
  renderBody: (text: string) => ReactNode;
}

/** `@@recall` — 질문만 보이고, 떠올린 뒤 답을 열어 한 번 눌러 스스로 채점한다. 채점하면 하루 뒤 복습에 들어간다. */
export function RecallPanel({ anchor, text, renderBody }: RecallPanelProps) {
  const t = useT(messages);
  const ctx = useExerciseContext();
  const { busy, send } = useSend(anchor);
  const [open, setOpen] = useState(false);
  const { question, key } = parseRecall(text);
  const keyView = key && (
    <div className="mt-2 border-l-2 border-line-strong pl-3">
      <p className="text-xs font-semibold text-fg-secondary">{t.recallKey}</p>
      {renderBody(key)}
    </div>
  );
  if (!ctx || !anchor) {
    return (
      <>
        {renderBody(question)}
        {keyView && <details className="mt-2"><summary className="cursor-pointer text-sm font-medium text-accent-text">{t.recallKey}</summary>{keyView}</details>}
      </>
    );
  }
  const hash = blockContentHash(text, "");
  const attempt = findAttempt(ctx.attempts, anchor, hash);
  if (attempt) {
    return (
      <>
        {renderBody(question)}
        {attempt.verdict !== "skipped" && keyView}
        {attempt.verdict !== "skipped" && (
          <p role="status" className="mt-2 text-sm text-fg-secondary">
            <span className="font-semibold text-fg">{t.recallGrade(attempt.verdict)}</span>
            {attempt.reviewItemId && ` · ${t.recallQueued}`}
          </p>
        )}
      </>
    );
  }
  return (
    <>
      {renderBody(question)}
      {!open ? (
        <div className="mt-2 flex flex-wrap items-center gap-2">
          <Button size="sm" onClick={() => setOpen(true)}>{t.recallReveal}</Button>
          <Button size="sm" variant="ghost" disabled={busy} onClick={() => send({ anchor, contentHash: hash, skip: true })}>{t.recallSkip}</Button>
        </div>
      ) : (
        <>
          {keyView}
          <p className="mt-2 text-sm text-fg-secondary">{t.recallHowMuch}</p>
          <div className="mt-1 flex flex-wrap items-center gap-2">
            {RECALL_GRADES.map((g) => (
              <Button key={g} size="sm" variant="secondary" disabled={busy} onClick={() => send({ anchor, contentHash: hash, verdict: g })}>
                {t.recallGrade(g)}
              </Button>
            ))}
          </div>
        </>
      )}
      <SubmitError anchor={anchor} />
    </>
  );
}

/** 요약을 떠올렸거나 건너뛰었는가(문맥이 없는 미리보기는 늘 true) — 요약 뒤 블록(다음 개념)을 보일지. */
export function useSummaryRecalled(anchor: string, text: string): boolean {
  const ctx = useExerciseContext();
  if (!ctx || !anchor) { return true; }
  return findAttempt(ctx.attempts, anchor, blockContentHash(text, "")) !== undefined;
}

interface SummaryRecallProps {
  anchor: string;
  text: string;
  summary: ReactNode;
}

/** `@@summary` — 먼저 핵심을 떠올려 적은 뒤(건너뛰기 가능) 요약을 연다. 적은 글은 저장하지 않고 적었는지만 남긴다. */
export function SummaryRecall({ anchor, text, summary }: SummaryRecallProps) {
  const t = useT(messages);
  const ctx = useExerciseContext();
  const { busy, send } = useSend(anchor);
  const [draft, setDraft] = useState("");
  if (!ctx || !anchor) { return <>{summary}</>; }
  const hash = blockContentHash(text, "");
  if (findAttempt(ctx.attempts, anchor, hash)) { return <>{summary}</>; }
  return (
    <div className="mt-2 flex flex-col gap-2">
      <Textarea
        aria-label={t.summaryRecallLabel}
        placeholder={t.summaryRecallPlaceholder}
        rows={3}
        value={draft}
        onChange={(event) => setDraft(event.target.value)}
        className="resize-y"
      />
      <div className="flex flex-wrap items-center gap-2">
        <Button size="sm" loading={busy} disabled={!draft.trim() || busy} onClick={() => send({ anchor, contentHash: hash, answer: draft })}>
          {t.summaryReveal}
        </Button>
        <Button size="sm" variant="ghost" disabled={busy} onClick={() => send({ anchor, contentHash: hash, skip: true })}>
          {t.summarySkip}
        </Button>
      </div>
      <SubmitError anchor={anchor} />
    </div>
  );
}
