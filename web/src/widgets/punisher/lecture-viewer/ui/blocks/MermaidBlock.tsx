
import { useEffect, useId, useState } from "react";
import { Skeleton } from "@/src/shared/ui/kit";
import { useResolvedTheme } from "@/src/entities/punisher";
import { CodeBlock } from "./CodeBlock";

let renderQueue: Promise<unknown> = Promise.resolve();

/** Mermaid renders through shared global state; serialize renders. */
function enqueue<T>(task: () => Promise<T>): Promise<T> {
  const next = renderQueue.then(task, task);
  renderQueue = next.catch(() => undefined);
  return next;
}

/** Mermaid diagram rendered client-side (mermaid lazy-loaded, theme-aware). */
export function MermaidBlock({ source, title }: { source: string; title?: string }) {
  const theme = useResolvedTheme();
  const rawId = useId();
  const [svg, setSvg] = useState<string | null>(null);
  const [failed, setFailed] = useState(false);

  useEffect(() => {
    let cancelled = false;
    const id = `mmd-${rawId.replace(/[^a-zA-Z0-9]/g, "")}-${theme}`;
    enqueue(async () => {
      const mermaid = (await import("mermaid")).default;
      mermaid.initialize({
        startOnLoad: false,
        securityLevel: "strict",
        theme: theme === "dark" ? "dark" : "default",
        // 이 옵션이 없으면 문법 오류 시 mermaid 가 "Syntax error" 그림을 <body>에 붙여 모든 화면에 남는다.
        suppressErrorRendering: true,
      });
      try {
        return await mermaid.render(id, source);
      } finally {
        // render() 가 예외를 던지면 임시 컨테이너가 남으므로 지운다.
        document.getElementById(`d${id}`)?.remove();
      }
    })
      .then(({ svg: result }) => {
        if (!cancelled) {
          setSvg(result);
          setFailed(false);
        }
      })
      .catch((err: unknown) => {
        console.warn("punisher: mermaid render failed", err);
        if (!cancelled) { setFailed(true); }
      });
    return () => {
      cancelled = true;
    };
  }, [source, theme, rawId]);

  if (failed) {
    return (
      <div>
        <p className="text-xs text-warning">도식을 그리지 못해 원문을 표시합니다.</p>
        <CodeBlock code={source} lang="mermaid" />
      </div>
    );
  }

  return (
    <figure
      role="img"
      aria-label={title ?? "도식"}
      className="my-4 flex justify-center overflow-x-auto rounded-md border border-line bg-canvas-elevated p-3"
    >
      {svg ? (
        // SVG is produced by mermaid with securityLevel "strict" (sanitized labels).
        <div className="max-w-full [&_svg]:h-auto [&_svg]:max-w-full" dangerouslySetInnerHTML={{ __html: svg }} />
      ) : (
        <Skeleton variant="block" className="h-40" />
      )}
    </figure>
  );
}
