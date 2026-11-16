import { AlertOctagon, ExternalLink } from "lucide-react";
import type { DocumentLink, LectureBlock } from "@/src/shared/api/punisher";
import { Markdown, type RenderTerm } from "@/src/shared/ui/punisher/Markdown";
import { TermPopover, type DeeperOrigin } from "@/src/features/punisher/term-popover";
import { closeUnclosedFence } from "../../lib/fences";
import { toInlineTermLinks } from "../../lib/inlineTerms";
import type { BlockGroup } from "../../lib/blockGroups";
import { splitMermaid } from "../../lib/mermaidInMarkdown";
import { CodeBlock } from "./CodeBlock";
import { MermaidBlock } from "./MermaidBlock";
import { StructuredBlock } from "./StructuredBlock";
import { AnswerBlock, ExerciseBlock, KeypointBlock, LabBlock, PauseBlock, SummaryBlock, WARNING_BOX } from "./StudyBlocks";
import { labForOS, useExerciseContext } from "@/src/features/punisher/lecture-exercise";
import { keepLineBreaks } from "../../lib/lineBreaks";

/** 링크 글자의 마지막 낱말과 아이콘을 한 덩어리로 묶어 아이콘만 다음 줄로 떨어지지 않게 한다. */
function SourceLabel({ text }: { text: string }) {
  const cut = text.lastIndexOf(" ") + 1;
  const tail = text.slice(cut);
  const icon = <ExternalLink className="ml-0.5 inline size-3 align-[-0.1em]" aria-hidden="true" />;
  // 공백 없는 긴 URL 은 줄바꿈이 필요하므로 묶지 않고 단어 결합 문자(U+2060)만 둔다.
  if (tail.length > 24) {
    return (
      <>
        {text}
        {"\u2060"}
        {icon}
      </>
    );
  }
  return (
    <>
      {text.slice(0, cut)}
      <span className="whitespace-nowrap">
        {tail}
        {icon}
      </span>
    </>
  );
}

export function SourcesList({ sources, title = "출처" }: { sources: DocumentLink[]; title?: string }) {
  if (sources.length === 0) { return null; }
  return (
    <aside className="my-4 border-t border-line pt-3 text-sm">
      <p className="mb-1 text-xs font-semibold text-fg-secondary">{title}</p>
      <ul className="flex flex-col gap-1">
        {sources.map((s) => (
          <li key={s.url}>
            <a
              href={s.url}
              target="_blank"
              rel="noreferrer noopener"
              className="inline break-words text-accent-text underline underline-offset-2"
            >
              <SourceLabel text={s.title || s.url} />
            </a>
          </li>
        ))}
      </ul>
    </aside>
  );
}

interface BlockRendererProps {
  block: LectureBlock;
  /** Omitted in the streaming preview (no interactions while generating). */
  onBeforeDeeper?: () => DeeperOrigin | void;
}

// 본문 문장 안의 [용어](term:개념id) → 누르면 용어 설명(스트리밍 미리보기에선 글자만).
function termRenderer(onBeforeDeeper: BlockRendererProps["onBeforeDeeper"]): RenderTerm | undefined {
  if (!onBeforeDeeper) { return undefined; }
  return function renderTerm(conceptId, label) {
    return <TermPopover inline term={String(label)} conceptId={conceptId} onBeforeDeeper={onBeforeDeeper} />;
  };
}

/** 묶인 블록(pause+출력, exercise+answer) — 묶이지 않은 블록은 BlockRenderer 그대로. */
export function BlockGroupRenderer({ group, onBeforeDeeper }: { group: BlockGroup; onBeforeDeeper?: BlockRendererProps["onBeforeDeeper"] }) {
  if (group.type === "pause") {
    const output = group.output ? <BlockRenderer block={group.output} onBeforeDeeper={onBeforeDeeper} /> : undefined;
    return <PauseBlock question={group.pause.text} output={output} renderTerm={termRenderer(onBeforeDeeper)} />;
  }
  if (group.type === "exercise") {
    return (
      <ExerciseBlock
        anchor={group.exercise.anchor}
        text={group.exercise.text}
        answer={group.answer ? (group.answer.text ?? "") : undefined}
        renderTerm={termRenderer(onBeforeDeeper)}
      />
    );
  }
  return <BlockRenderer block={group.block} onBeforeDeeper={onBeforeDeeper} />;
}

export function BlockRenderer({ block, onBeforeDeeper }: BlockRendererProps) {
  const exercise = useExerciseContext();
  const text = block.text ?? "";
  const renderTerm = termRenderer(onBeforeDeeper);
  switch (block.type) {
    case "markdown":
      return (
        <>
          {splitMermaid(toInlineTermLinks(closeUnclosedFence(text))).map((part, index) =>
            part.kind === "mermaid" ? (
              <MermaidBlock key={index} source={part.text} />
            ) : (
              <Markdown key={index} renderTerm={renderTerm}>{part.text}</Markdown>
            ),
          )}
        </>
      );
    case "code":
      return <CodeBlock code={text} lang={block.lang} />;
    case "log":
      return <CodeBlock code={text} lang={block.lang} isLog />;
    case "mermaid":
      return <MermaidBlock source={text} />;
    case "misconception":
      return (
        <aside role="note" className={WARNING_BOX}>
          <p className="flex items-center gap-1.5 text-sm font-semibold text-warning">
            <AlertOctagon className="size-4" aria-hidden="true" />
            흔한 오해
          </p>
          {/* 원문의 ①②③ 은 줄바꿈 하나로 나뉘어 있다 — 이 블록에서만 줄바꿈을 살린다. */}
          <Markdown renderTerm={renderTerm}>{keepLineBreaks(toInlineTermLinks(closeUnclosedFence(text)))}</Markdown>
        </aside>
      );
    case "term":
      if (!block.conceptId || !onBeforeDeeper) {
        return (
          <p className="my-3">
            <strong className="font-semibold">{block.term}</strong> {text}
          </p>
        );
      }
      return (
        <div className="my-1">
          <TermPopover
            term={block.term ?? block.conceptId}
            conceptId={block.conceptId}
            fallbackText={text}
            proposed={block.proposed}
            onBeforeDeeper={onBeforeDeeper}
          />
        </div>
      );
    case "sources":
      return <SourcesList sources={block.sources ?? []} />;
    case "keypoint":
      return <KeypointBlock kind={block.kind} text={text} renderTerm={renderTerm} />;
    case "trace":
    case "chart":
    case "diff":
      return <StructuredBlock block={block} />;
    case "pause":
      // 다음 출력 블록과 묶는 일은 LectureBody(groupBlocks)가 한다. 여기서는 짝 없는 질문만.
      return <PauseBlock question={text} renderTerm={renderTerm} />;
    case "exercise":
      return <ExerciseBlock anchor={block.anchor} text={text} renderTerm={renderTerm} />;
    case "lab":
      // 강의 화면에서는 내 실습 환경(설정)에 맞는 블록만, 미리보기에선 모두.
      if (exercise && !labForOS(block.kind, exercise.labOs)) { return null; }
      return <LabBlock anchor={block.anchor} text={text} />;
    case "answer":
      return <AnswerBlock text={text} renderTerm={renderTerm} />;
    case "summary":
      return <SummaryBlock text={text} renderTerm={renderTerm} />;
    default:
      // 알 수 없는 블록 타입(새 서버·옛 화면)은 그리지 않는다.
      return null;
  }
}
