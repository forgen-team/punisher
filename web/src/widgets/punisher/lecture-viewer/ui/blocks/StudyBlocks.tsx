
import type { ReactNode } from "react";
import { AlertTriangle, BookMarked, ListChecks, NotebookPen, ScrollText, Sparkles, type LucideIcon } from "lucide-react";
import { cn } from "@/src/shared/ui/kit";
import { Markdown, type RenderTerm } from "@/src/shared/ui/punisher/Markdown";
import { closeUnclosedFence } from "../../lib/fences";
import { toInlineTermLinks } from "../../lib/inlineTerms";
import { BLOCK_LABELS, KEYPOINT_FALLBACK_LABEL, KEYPOINT_LABELS } from "./constants";
import { Reveal } from "./Reveal";

function Body({ text, renderTerm }: { text?: string; renderTerm?: RenderTerm }) {
  return <Markdown renderTerm={renderTerm}>{toInlineTermLinks(closeUnclosedFence(text ?? ""))}</Markdown>;
}

const KEYPOINT_ICONS: Record<string, LucideIcon> = { definition: BookMarked, caution: AlertTriangle, rule: ListChecks };

function CardTitle({ icon: Icon, className, children }: { icon: LucideIcon; className?: string; children: ReactNode }) {
  return (
    <p className={cn("flex items-center gap-1.5 text-sm font-semibold", className)}>
      <Icon className="size-4 shrink-0" aria-hidden="true" />
      {children}
    </p>
  );
}

/** `@@keypoint kind=…` — 강조 상자(제목은 종류 라벨) */
export function KeypointBlock({ kind, text, renderTerm }: { kind?: string; text?: string; renderTerm?: RenderTerm }) {
  const caution = kind === "caution";
  const icon = KEYPOINT_ICONS[kind ?? ""] ?? Sparkles;
  return (
    <aside
      role="note"
      className={cn("my-4 rounded-md border-l-2 px-3 py-2", caution ? "border-warning bg-warning-subtle" : "border-accent bg-accent-subtle")}
    >
      <CardTitle icon={icon} className={caution ? "text-warning" : "text-accent-text"}>
        {KEYPOINT_LABELS[kind ?? ""] ?? KEYPOINT_FALLBACK_LABEL}
      </CardTitle>
      <Body text={text} renderTerm={renderTerm} />
    </aside>
  );
}

/** `@@pause` — 질문 + 다음 출력 블록을 접어 둔다('출력 보기') */
export function PauseBlock({ question, output }: { question?: string; output?: ReactNode }) {
  return (
    <div className="my-4 rounded-md border border-line bg-canvas-elevated px-3 py-2">
      <CardTitle icon={NotebookPen} className="text-fg">{BLOCK_LABELS.pause}</CardTitle>
      {question && <p className="mt-1">{question}</p>}
      {output && <Reveal label={BLOCK_LABELS.showOutput}>{output}</Reveal>}
    </div>
  );
}

/** `@@exercise` (+ 바로 다음 `@@answer`) — 문제 카드 + 접힌 풀이 */
export function ExerciseBlock({ text, answer, renderTerm }: { text?: string; answer?: string; renderTerm?: RenderTerm }) {
  return (
    <section className="my-4 rounded-md border border-line bg-canvas-elevated px-3 py-2">
      <CardTitle icon={NotebookPen} className="text-fg">{BLOCK_LABELS.exercise}</CardTitle>
      <Body text={text} renderTerm={renderTerm} />
      {answer !== undefined && (
        <Reveal label={BLOCK_LABELS.showAnswer}>
          <div className="border-l-2 border-line-strong pl-3">
            <Body text={answer} renderTerm={renderTerm} />
          </div>
        </Reveal>
      )}
    </section>
  );
}

/** 짝 없는 `@@answer` — 접힌 풀이만 */
export function AnswerBlock({ text, renderTerm }: { text?: string; renderTerm?: RenderTerm }) {
  return (
    <div className="my-3">
      <Reveal label={BLOCK_LABELS.showAnswer}>
        <div className="border-l-2 border-line-strong pl-3">
          <Body text={text} renderTerm={renderTerm} />
        </div>
      </Reveal>
    </div>
  );
}

/** `@@summary` — 강의 끝 한 장 요약 카드 */
export function SummaryBlock({ text, renderTerm }: { text?: string; renderTerm?: RenderTerm }) {
  return (
    <section aria-label={BLOCK_LABELS.summary} className="my-6 rounded-xl border border-line bg-canvas-elevated px-4 py-3 shadow-xs">
      <CardTitle icon={ScrollText} className="text-fg">{BLOCK_LABELS.summary}</CardTitle>
      <Body text={text} renderTerm={renderTerm} />
    </section>
  );
}
