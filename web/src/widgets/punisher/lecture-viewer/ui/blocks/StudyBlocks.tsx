import type { ReactNode } from "react";
import { AlertTriangle, BookMarked, Brain, ListChecks, NotebookPen, ScrollText, Sparkles, Terminal, type LucideIcon } from "lucide-react";
import { cn } from "@/src/shared/ui/kit";
import { Markdown, type RenderTerm } from "@/src/shared/ui/punisher/Markdown";
import { ExercisePanel, LabResultPanel, parseLab, PausePanel, RecallPanel, SummaryRecall } from "@/src/features/punisher/lecture-exercise";
import { CodeBlock } from "./CodeBlock";
import { closeUnclosedFence } from "../../lib/fences";
import { toInlineTermLinks } from "../../lib/inlineTerms";
import { useT } from "@/src/shared/i18n";
import { blockMessages } from "../../i18n";
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

/** '주의'·'흔한 오해' 상자(경고색) — 규칙·정의(액센트 막대 + 옅은 회색 면)와 다크에서도 구분되게 면 색을 나눈다. */
export const WARNING_BOX = "my-4 rounded-md border-l-2 border-warning bg-warning-subtle px-3 py-2";

/** `@@keypoint kind=…` — 강조 상자(제목은 종류 라벨) */
export function KeypointBlock({ kind, text, renderTerm }: { kind?: string; text?: string; renderTerm?: RenderTerm }) {
  const caution = kind === "caution";
  const t = useT(blockMessages);
  const icon = KEYPOINT_ICONS[kind ?? ""] ?? Sparkles;
  return (
    <aside
      role="note"
      className={caution ? WARNING_BOX : "my-4 rounded-md border-l-2 border-accent bg-canvas-subtle px-3 py-2"}
    >
      <CardTitle icon={icon} className={caution ? "text-warning" : "text-accent-text"}>
        {t.keypoint(kind ?? "")}
      </CardTitle>
      <Body text={text} renderTerm={renderTerm} />
    </aside>
  );
}

interface PauseBlockProps {
  anchor?: string;
  question?: string;
  output?: ReactNode;
  /** 접어 둔 출력 블록의 원문(시도 기록의 content hash) */
  outputText?: string;
  renderTerm?: RenderTerm;
}

/** `@@pause` — 질문(본문과 같은 Markdown·용어 링크) + 다음 출력 블록. 강의 화면에서는 예측·확신도를 남긴 뒤 열고
 * 같았는지 한 번 누른다. 미리보기에선 '출력 보기'로 접어 둔다. */
export function PauseBlock({ anchor, question, output, outputText, renderTerm }: PauseBlockProps) {
  const t = useT(blockMessages);
  const folded = output && <Reveal label={t.showOutput}>{output}</Reveal>;
  return (
    <div className="my-4 rounded-md border border-line bg-canvas-elevated px-3 py-2">
      <CardTitle icon={NotebookPen} className="text-fg">{t.pause}</CardTitle>
      {question && <Body text={question} renderTerm={renderTerm} />}
      {output && (
        <PausePanel anchor={anchor ?? ""} question={question ?? ""} outputText={outputText ?? ""} output={output} fallback={folded} />
      )}
    </div>
  );
}

/** `@@recall` — 섹션 끝 '방금 읽은 걸 떠올려 보기'. 질문만 보이고, 답은 떠올린 뒤 연다. */
export function RecallBlock({ anchor, text, renderTerm }: { anchor?: string; text?: string; renderTerm?: RenderTerm }) {
  const t = useT(blockMessages);
  return (
    <section className="my-4 rounded-md border border-dashed border-line-strong bg-canvas-subtle px-3 py-2">
      <CardTitle icon={Brain} className="text-fg">{t.recall}</CardTitle>
      <RecallPanel anchor={anchor ?? ""} text={text ?? ""} renderBody={(body) => <Body text={body} renderTerm={renderTerm} />} />
    </section>
  );
}

/** `@@exercise` (+ 바로 다음 `@@answer`) — 문제 카드. 강의 화면에서는 답을 써야 풀이가 열리고(건너뛰기 가능)
 * 튜터 평이 뒤따른다. 미리보기에선 접힌 풀이. */
export function ExerciseBlock({ anchor, kind, text, answer, renderTerm }: { anchor?: string; kind?: string; text?: string; answer?: string; renderTerm?: RenderTerm }) {
  const t = useT(blockMessages);
  const kindLabel = t.exerciseKind(kind ?? "");
  const folded = answer !== undefined && (
    <Reveal label={t.showAnswer}>
      <div className="border-l-2 border-line-strong pl-3">
        <Body text={answer} renderTerm={renderTerm} />
      </div>
    </Reveal>
  );
  return (
    <section className="my-4 rounded-md border border-line bg-canvas-elevated px-3 py-2">
      <CardTitle icon={NotebookPen} className="text-fg">
        {t.exercise}
        {kindLabel && <span className="ml-1 rounded-sm bg-canvas-subtle px-1.5 text-xs font-medium text-fg-secondary">{kindLabel}</span>}
      </CardTitle>
      <Body text={text} renderTerm={renderTerm} />
      {answer !== undefined && (
        <ExercisePanel
          anchor={anchor ?? ""}
          problem={text ?? ""}
          solution={answer}
          solutionView={<Body text={answer} renderTerm={renderTerm} />}
          fallback={folded}
        />
      )}
    </section>
  );
}

/** `@@lab os=…` — 내 터미널에서 실행할 명령(복사) · 보는 곳 · 예상, 결과를 붙여 넣으면 튜터가 읽어 준다. */
export function LabBlock({ anchor, text }: { anchor?: string; text?: string }) {
  const t = useT(blockMessages);
  const lab = parseLab(text ?? "");
  return (
    <section className="my-4 rounded-md border border-line bg-canvas-elevated px-3 py-2">
      <CardTitle icon={Terminal} className="text-fg">{t.lab}</CardTitle>
      <CodeBlock code={lab.command} lang="bash" label={t.labCommand} />
      <dl className="mt-1 grid grid-cols-[auto_1fr] gap-x-3 gap-y-1 text-sm">
        {lab.look && (
          <>
            <dt className="font-medium text-fg-secondary">{t.labLook}</dt>
            <dd className="break-keep">{lab.look}</dd>
          </>
        )}
        {lab.expect && (
          <>
            <dt className="font-medium text-fg-secondary">{t.labExpect}</dt>
            <dd className="break-keep">{lab.expect}</dd>
          </>
        )}
      </dl>
      <LabResultPanel anchor={anchor ?? ""} text={text ?? ""} />
    </section>
  );
}

/** 짝 없는 `@@answer` — 접힌 풀이만 */
export function AnswerBlock({ text, renderTerm }: { text?: string; renderTerm?: RenderTerm }) {
  const t = useT(blockMessages);
  return (
    <div className="my-3">
      <Reveal label={t.showAnswer}>
        <div className="border-l-2 border-line-strong pl-3">
          <Body text={text} renderTerm={renderTerm} />
        </div>
      </Reveal>
    </div>
  );
}

/** `@@summary` — 강의 끝 한 장 요약 카드. 강의 화면에서는 핵심을 먼저 떠올려 적은 뒤(건너뛰기 가능) 연다. */
export function SummaryBlock({ anchor, text, renderTerm }: { anchor?: string; text?: string; renderTerm?: RenderTerm }) {
  const t = useT(blockMessages);
  return (
    <section aria-label={t.summary} className="my-6 rounded-xl border border-line bg-canvas-elevated px-4 py-3 shadow-xs">
      <CardTitle icon={ScrollText} className="text-fg">{t.summary}</CardTitle>
      <SummaryRecall anchor={anchor ?? ""} text={text ?? ""} summary={<Body text={text} renderTerm={renderTerm} />} />
    </section>
  );
}
