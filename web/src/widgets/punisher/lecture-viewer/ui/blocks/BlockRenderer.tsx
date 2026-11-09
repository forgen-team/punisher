
import { AlertOctagon, ExternalLink } from "lucide-react";
import type { DocumentLink, LectureBlock } from "@/src/shared/api/punisher";
import { Markdown } from "@/src/shared/ui/punisher/Markdown";
import { TermPopover, type DeeperOrigin } from "@/src/features/punisher/term-popover";
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

/** 모델이 문장 중간에 쓴 "@@term 용어"(지시어는 줄 맨 앞만 유효)는 지시어만 지우고 용어는 남긴다. */
function stripInlineDirectives(text: string): string {
  return text.replace(/@@term\s+/g, "");
}

export function BlockRenderer({ block, onBeforeDeeper }: BlockRendererProps) {
  const text = block.text ?? "";
  switch (block.type) {
    case "markdown":
      return <Markdown>{stripInlineDirectives(text)}</Markdown>;
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
          <Markdown>{stripInlineDirectives(text)}</Markdown>
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
