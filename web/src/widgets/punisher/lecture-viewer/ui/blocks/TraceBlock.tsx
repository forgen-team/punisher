
import { useId, useMemo, useState } from "react";
import { Button, cn } from "@/src/shared/ui/kit";
import { traceProseColumns, type TraceData } from "../../lib/blockData";
import { initialReveal, revealNext } from "../../lib/blockGroups";
import { useT } from "@/src/shared/i18n";
import { blockMessages } from "../../i18n";

/** `@@trace` — 손계산 표. '다음 단계'로 한 행씩 펼치고 '모두 보기'로 전부 연다. */
export function TraceBlock({ data }: { data: TraceData }) {
  const t = useT(blockMessages);
  const total = data.rows.length;
  const [shown, setShown] = useState(() => initialReveal(total));
  const tableId = useId();
  const done = shown >= total;
  const prose = useMemo(() => traceProseColumns(data), [data]);
  return (
    <figure className="my-4">
      {data.caption && <figcaption className="mb-2 text-sm font-semibold text-fg">{data.caption}</figcaption>}
      <div className="max-w-full overflow-x-auto" tabIndex={0} role="region" aria-label={data.caption || t.traceTable}>
        {/* 숫자 칸은 고정폭·한 줄, 글 칸은 본문 글꼴로 18rem 에서 줄바꿈 — 설명 열 때문에 표가 화면 밖으로 길어지지 않게. */}
        <table id={tableId} className="w-full border-collapse text-sm tabular-nums">
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
                  <td key={c} className={cn("border border-line px-2 py-1 align-top", !prose[c] && "whitespace-nowrap font-mono")}>
                    {prose[c] ? <div className="min-w-[8rem] max-w-[18rem] whitespace-normal">{value}</div> : value}
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
            {t.nextStep}
          </Button>
          <Button size="sm" variant="ghost" aria-controls={tableId} onClick={() => setShown(total)}>
            {t.showAll}
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
