
import { useMemo } from "react";
import type { DiffData } from "../../lib/blockData";
import { diffLines } from "../../lib/lineDiff";
import { CodeBlock } from "./CodeBlock";
import { BLOCK_LABELS } from "./constants";

/** `@@diff lang=…` — 좁은 폭은 위·아래, 넓은 폭(md+)은 좌·우. 바뀐 줄은 전(빨강)·후(초록)로 강조. */
export function DiffBlock({ data, lang }: { data: DiffData; lang?: string }) {
  const changes = useMemo(() => diffLines(data.before, data.after), [data.before, data.after]);
  const suffix = lang ? ` · ${lang}` : "";
  return (
    <div className="my-4">
      <div className="grid gap-0 md:grid-cols-2 md:gap-3 [&_figure]:my-2 md:[&_figure]:my-0">
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
