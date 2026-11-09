
import type { ReactNode } from "react";
import ReactMarkdown, { type Components } from "react-markdown";
import remarkGfm from "remark-gfm";
import { cn } from "@/src/shared/ui/kit";

const components: Components = {
  h1: ({ children }) => <h3 className="mb-2 mt-6 text-xl font-semibold text-fg">{children}</h3>,
  h2: ({ children }) => <h3 className="mb-2 mt-6 text-xl font-semibold text-fg">{children}</h3>,
  h3: ({ children }) => <h4 className="mb-2 mt-4 text-base font-semibold text-fg">{children}</h4>,
  p: ({ children }) => <p className="my-3">{children}</p>,
  ul: ({ children }) => <ul className="my-3 list-disc space-y-1 pl-6">{children}</ul>,
  ol: ({ children }) => <ol className="my-3 list-decimal space-y-1 pl-6">{children}</ol>,
  a: ({ children, href }) => (
    <a href={href} target="_blank" rel="noreferrer noopener" className="text-accent-text underline underline-offset-2">
      {children}
    </a>
  ),
  blockquote: ({ children }) => (
    <blockquote className="my-3 border-l-2 border-line-strong pl-3 text-fg-secondary">{children}</blockquote>
  ),
  code: ({ children, className }) => (
    <code className={cn("rounded-sm bg-canvas-subtle px-1 py-0.5 font-mono text-sm", className)}>{children}</code>
  ),
  pre: ({ children }) => (
    <pre className="my-3 overflow-x-auto rounded-md border border-line bg-canvas-subtle p-3 text-sm [&>code]:bg-transparent [&>code]:p-0">
      {children}
    </pre>
  ),
  table: ({ children }) => (
    <div className="my-3 overflow-x-auto">
      <table className="w-full border-collapse text-sm">{children}</table>
    </div>
  ),
  th: ({ children }) => <th className="border border-line bg-canvas-subtle px-2 py-1 text-left font-semibold">{children}</th>,
  td: ({ children }) => <td className="border border-line px-2 py-1">{children}</td>,
};

/**
 * GitHub-flavoured markdown with token-based styles (raw HTML is not rendered).
 * 본문 글자 크기·행간은 부모가 정한다(강의 본문 16px/1.7, 패널 14px).
 */
export function Markdown({ children, className }: { children: string; className?: string }): ReactNode {
  return (
    <div className={cn("break-words text-fg", className)}>
      <ReactMarkdown remarkPlugins={[remarkGfm]} components={components}>
        {children}
      </ReactMarkdown>
    </div>
  );
}
