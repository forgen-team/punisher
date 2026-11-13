
import { useId, useState } from "react";
import { Button } from "@/src/shared/ui/kit";
import type { TraceData } from "../../lib/blockData";
import { initialReveal, revealNext } from "../../lib/blockGroups";
import { BLOCK_LABELS } from "./constants";

/** `@@trace` — 손계산 표. '다음 단계'로 한 행씩 펼치고 '모두 보기'로 전부 연다. */
export function TraceBlock({ data }: { data: TraceData }) {
  const total = data.rows.length;
  const [shown, setShown] = useState(() => initialReveal(total));
  const tableId = useId();
  const done = shown >= total;
  return (
    <figure className="my-4">
      {data.caption && <figcaption className="mb-2 text-sm font-semibold text-fg">{data.caption}</figcaption>}
      <div className="max-w-full overflow-x-auto" tabIndex={0} role="region" aria-label={data.caption || BLOCK_LABELS.traceTable}>
        <table id={tableId} className="w-full min-w-max border-collapse text-sm tabular-nums">
          <thead>
            <tr>
              {data.columns.map((col, index) => (
                <th key={index} scope="col" className="border border-line bg-canvas-subtle px-2 py-1 text-left font-semibold">
                  {col}
                </th>
              ))}
            </tr>
          </thead>
          <tbody aria-live="polite">
            {data.rows.slice(0, shown).map((row, r) => (
              <tr key={r}>
                {row.map((value, c) => (
                  <td key={c} className="border border-line px-2 py-1 font-mono">
                    {value}
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      {!done && (
        <div className="mt-2 flex flex-wrap items-center gap-2">
          <Button size="sm" variant="secondary" aria-controls={tableId} onClick={() => setShown((n) => revealNext(n, total))}>
            {BLOCK_LABELS.nextStep}
          </Button>
          <Button size="sm" variant="ghost" aria-controls={tableId} onClick={() => setShown(total)}>
            {BLOCK_LABELS.showAll}
          </Button>
          <span className="text-xs text-fg-muted tabular-nums">
            {shown}/{total}
          </span>
        </div>
      )}
      {done && data.note && <p className="mt-2 text-sm text-fg-secondary">{data.note}</p>}
    </figure>
  );
}
