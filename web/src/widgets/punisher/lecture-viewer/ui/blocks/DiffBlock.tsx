
import { useEffect, useMemo, useRef, useState } from "react";
import { cn } from "@/src/shared/ui/kit";
import type { DiffData } from "../../lib/blockData";
import { diffLines } from "../../lib/lineDiff";
import { CodeBlock } from "./CodeBlock";
import { BLOCK_LABELS } from "./constants";

/** 본문 폭이 이 값 이상일 때만 전·후를 좌우로 둔다(한 칸이 약 370px 이상 — 화면 폭이 아니라 본문 칸 기준). */
const SIDE_BY_SIDE_MIN_WIDTH = 760;

/** 이 블록이 놓인 본문 칸의 폭(px). 처음 그릴 때는 0(위·아래 배치). */
function useBoxWidth() {
  const ref = useRef<HTMLDivElement>(null);
  const [width, setWidth] = useState(0);
  useEffect(() => {
    const el = ref.current;
    if (!el) { return undefined; }
    setWidth(el.clientWidth);
    if (typeof ResizeObserver === "undefined") { return undefined; }
    const observer = new ResizeObserver(([entry]) => setWidth(entry.contentRect.width));
    observer.observe(el);
    return () => observer.disconnect();
  }, []);
  return { ref, width };
}

/** `@@diff lang=…` — 본문 칸이 좁으면 위·아래, 넓으면(760px+) 좌·우(두 상자 높이 같게). 바뀐 줄은 전(빨강)·후(초록)로 강조. */
export function DiffBlock({ data, lang }: { data: DiffData; lang?: string }) {
  const changes = useMemo(() => diffLines(data.before, data.after), [data.before, data.after]);
  const { ref, width } = useBoxWidth();
  const side = width >= SIDE_BY_SIDE_MIN_WIDTH;
  const suffix = lang ? ` · ${lang}` : "";
  return (
    <div ref={ref} className="my-4">
      <div className={cn("grid", side ? "grid-cols-2 gap-3 [&_figure]:my-0 [&_figure]:h-full" : "gap-0 [&_figure]:my-2")}>
        <div className="min-w-0">
          <CodeBlock code={data.before} lang={lang} label={`${BLOCK_LABELS.diffBefore}${suffix}`} changedLines={changes.removed} changeTone="removed" />
        </div>
        <div className="min-w-0">
          <CodeBlock code={data.after} lang={lang} label={`${BLOCK_LABELS.diffAfter}${suffix}`} changedLines={changes.added} changeTone="added" />
        </div>
      </div>
      {data.note && <p className="mt-2 text-sm text-fg-secondary">{data.note}</p>}
    </div>
  );
}
