
import { useEffect, useState } from "react";
import { Copy } from "lucide-react";
import { cn } from "@/src/shared/ui/kit";
import { useT } from "@/src/shared/i18n";
import { blockMessages } from "../../i18n";

type ChangeTone = "removed" | "added";

interface CodeBlockProps {
  code: string;
  lang?: string;
  isLog?: boolean;
  /** 머리줄 이름(기본: 언어·로그) */
  label?: string;
  /** 강조할 줄(1부터, diff 블록) */
  changedLines?: readonly number[];
  changeTone?: ChangeTone;
}

const CHANGE_TONE: Record<ChangeTone, string> = {
  removed: "[&_.line-changed]:bg-danger-subtle [&_.line-changed]:shadow-[inset_3px_0_0_var(--color-danger)]",
  added: "[&_.line-changed]:bg-success-subtle [&_.line-changed]:shadow-[inset_3px_0_0_var(--color-success)]",
};
// 바뀐 줄 표시(막대)가 있으면 좌우 여백을 상자 대신 각 줄에 둔다 — 막대는 상자 가장자리, 글자는 막대에서 띄운다.
const MARKED_LINES = "px-0 [&_.line]:px-3 [&_.line-changed]:pl-4";
const NO_LINES: readonly number[] = [];

/** Code/log block; shiki is lazy-loaded and themed via CSS variables (light/dark). */
export function CodeBlock({ code, lang, isLog, label, changedLines = NO_LINES, changeTone = "added" }: CodeBlockProps) {
  const t = useT(blockMessages);
  const [html, setHtml] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);

  useEffect(() => {
    let cancelled = false;
    import("../../lib/highlight")
      .then(({ highlightToHtml }) => highlightToHtml(code, isLog ? "log" : lang, changedLines))
      .then((result) => {
        if (!cancelled) { setHtml(result); }
      })
      .catch((err: unknown) => console.warn("punisher: highlight failed", err));
    return () => {
      cancelled = true;
    };
  }, [code, lang, isLog, changedLines]);

  async function copy() {
    try {
      await navigator.clipboard.writeText(code);
      setCopied(true);
      setTimeout(() => setCopied(false), 1500);
    } catch (err) {
      console.warn("punisher: clipboard failed", err);
    }
  }

  const heading = label ?? (isLog ? `${t.log}${lang ? ` · ${lang}` : ""}` : lang || t.code);
  const marking = changedLines.length > 0;

  return (
    <figure className="punisher-code my-4 overflow-hidden rounded-md border border-line bg-canvas-subtle">
      <figcaption className="flex items-center justify-between border-b border-line pl-3 text-xs text-fg-muted">
        <span>{heading}</span>
        <button
          type="button"
          onClick={copy}
          className="inline-flex min-h-9 items-center gap-1 px-3 hover:text-fg"
          aria-label={copied ? t.copied : t.copyCode}
        >
          <Copy className="size-3.5" aria-hidden="true" />
          <span aria-hidden="true">{copied ? t.copied : t.copy}</span>
        </button>
      </figcaption>
      {html ? (
        <div
          className={cn(
            "overflow-x-auto p-3 text-sm leading-6 [&_pre]:!bg-transparent [&_span]:text-[var(--shiki-light)] [[data-theme=dark]_&_span]:text-[var(--shiki-dark)]",
            marking && "[&_.line]:inline-block [&_.line]:min-w-full",
            marking && MARKED_LINES,
            marking && CHANGE_TONE[changeTone],
          )}
          // shiki output is generated locally from text tokens (escaped), not from raw HTML input.
          dangerouslySetInnerHTML={{ __html: html }}
        />
      ) : (
        <pre className={cn("overflow-x-auto p-3 text-sm leading-6 text-fg", marking && MARKED_LINES, marking && CHANGE_TONE[changeTone])}>
          <code>
            {marking
              ? code.split("\n").map((line, index) => (
                  <span key={index} className={cn("line block min-w-full", changedLines.includes(index + 1) && "line-changed")}>
                    {line || " "}
                  </span>
                ))
              : code}
          </code>
        </pre>
      )}
    </figure>
  );
}
