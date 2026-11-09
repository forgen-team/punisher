
import { AlertOctagon, ExternalLink } from "lucide-react";
import type { DocumentLink, LectureBlock } from "@/src/shared/api/punisher";
import { Markdown, type RenderTerm } from "@/src/shared/ui/punisher/Markdown";
import { TermPopover, type DeeperOrigin } from "@/src/features/punisher/term-popover";
import { closeUnclosedFence } from "../../lib/fences";
import { toInlineTermLinks } from "../../lib/inlineTerms";
import { splitMermaid } from "../../lib/mermaidInMarkdown";
import { CodeBlock } from "./CodeBlock";
import { MermaidBlock } from "./MermaidBlock";

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
              className="inline-flex items-center gap-1 break-all text-accent-text underline underline-offset-2"
            >
              {s.title || s.url}
              <ExternalLink className="size-3 shrink-0" aria-hidden="true" />
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

export function BlockRenderer({ block, onBeforeDeeper }: BlockRendererProps) {
  const text = block.text ?? "";
  // 본문 문장 안의 [용어](term:개념id) → 누르면 용어 설명(스트리밍 미리보기에선 글자만).
  const renderTerm: RenderTerm | undefined = onBeforeDeeper
    ? (conceptId, label) => (
        <TermPopover inline term={String(label)} conceptId={conceptId} onBeforeDeeper={onBeforeDeeper} />
      )
    : undefined;
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
        <aside role="note" className="my-4 border-l-2 border-warning bg-warning-subtle px-3 py-2">
          <p className="flex items-center gap-1.5 text-sm font-semibold text-warning">
            <AlertOctagon className="size-4" aria-hidden="true" />
            흔한 오해
          </p>
          <Markdown renderTerm={renderTerm}>{toInlineTermLinks(closeUnclosedFence(text))}</Markdown>
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
    default:
      return null;
  }
}
