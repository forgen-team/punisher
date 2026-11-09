
import { useEffect, useState } from "react";
import { Copy } from "lucide-react";

interface CodeBlockProps {
  code: string;
  lang?: string;
  isLog?: boolean;
}

/** Code/log block; shiki is lazy-loaded and themed via CSS variables (light/dark). */
export function CodeBlock({ code, lang, isLog }: CodeBlockProps) {
  const [html, setHtml] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);

  useEffect(() => {
    let cancelled = false;
    import("../../lib/highlight")
      .then(({ highlightToHtml }) => highlightToHtml(code, isLog ? "log" : lang))
      .then((result) => {
        if (!cancelled) { setHtml(result); }
      })
      .catch((err: unknown) => console.warn("punisher: highlight failed", err));
    return () => {
      cancelled = true;
    };
  }, [code, lang, isLog]);

  async function copy() {
    try {
      await navigator.clipboard.writeText(code);
      setCopied(true);
      setTimeout(() => setCopied(false), 1500);
    } catch (err) {
      console.warn("punisher: clipboard failed", err);
    }
  }

  const label = isLog ? `로그${lang ? ` · ${lang}` : ""}` : lang || "코드";

  return (
    <figure className="punisher-code my-4 overflow-hidden rounded-md border border-line bg-canvas-subtle">
      <figcaption className="flex items-center justify-between border-b border-line pl-3 text-xs text-fg-muted">
        <span>{label}</span>
        <button
          type="button"
          onClick={copy}
          className="inline-flex min-h-9 items-center gap-1 px-3 hover:text-fg"
          aria-label={copied ? "복사됨" : "코드 복사"}
        >
          <Copy className="size-3.5" aria-hidden="true" />
          <span aria-hidden="true">{copied ? "복사됨" : "복사"}</span>
        </button>
      </figcaption>
      {html ? (
        <div
          className="overflow-x-auto p-3 text-sm leading-6 [&_pre]:!bg-transparent [&_span]:text-[var(--shiki-light)] [[data-theme=dark]_&_span]:text-[var(--shiki-dark)]"
          // shiki output is generated locally from text tokens (escaped), not from raw HTML input.
          dangerouslySetInnerHTML={{ __html: html }}
        />
      ) : (
        <pre className="overflow-x-auto p-3 text-sm leading-6 text-fg">
          <code>{code}</code>
        </pre>
      )}
    </figure>
  );
}
