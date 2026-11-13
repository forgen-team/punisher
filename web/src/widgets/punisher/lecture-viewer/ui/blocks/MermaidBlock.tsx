
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

function tokenColor(name: string): string {
  return getComputedStyle(document.documentElement).getPropertyValue(name).trim();
}

/** 값이 있는 토큰만 mermaid 변수로 넘긴다(토큰을 못 읽으면 mermaid 기본값). */
function tokenVariables(map: Record<string, string>): Record<string, string> {
  return Object.fromEntries(Object.entries(map).flatMap(([key, token]) => {
    const value = tokenColor(token);
    return value ? [[key, value]] : [];
  }));
}

/** 다크 모드 간선 라벨: 기본(어두운 회색 반투명) 위 글자 대비가 낮아, 도식 면 색과 본문 글자 색으로 맞춘다. */
function darkThemeVariables(): Record<string, string> {
  return tokenVariables({ edgeLabelBackground: "--color-bg-elevated", textColor: "--color-text" });
}

/** 라이트 모드: 기본 테마의 연보라 대신 앱 토큰(액센트 옅은 면·액센트 테두리·본문 글자)으로 그린다. */
function lightThemeVariables(): Record<string, string> {
  return tokenVariables({
    primaryColor: "--color-accent-subtle",
    primaryBorderColor: "--color-accent",
    primaryTextColor: "--color-text",
    secondaryColor: "--color-bg-subtle",
    tertiaryColor: "--color-bg-elevated",
    lineColor: "--color-text-secondary",
    textColor: "--color-text",
    edgeLabelBackground: "--color-bg-elevated",
    clusterBkg: "--color-bg-subtle",
    clusterBorder: "--color-border-strong",
  });
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
        theme: theme === "dark" ? "dark" : "base",
        themeVariables: theme === "dark" ? darkThemeVariables() : lightThemeVariables(),
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
        // w-full: 가운데 정렬 flex 안에서 폭이 내용에 맞춰 줄면 SVG(width=100%)가 함께 줄어 글자가 작아진다.
        <div className="w-full [&_svg]:mx-auto [&_svg]:h-auto [&_svg]:max-w-full" dangerouslySetInnerHTML={{ __html: svg }} />
      ) : (
        <Skeleton variant="block" className="h-40" />
      )}
    </figure>
  );
}
