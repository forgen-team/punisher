
import { useEffect, useRef, useState } from "react";
import type { ChartData } from "../../lib/blockData";
import { barRect, formatTick, formatValue, layoutChart, linePath } from "../../lib/chartScale";
import { useT } from "@/src/shared/i18n";
import { blockMessages } from "../../i18n";

// 색은 토큰만 쓴다. 색만으로 구분하지 않도록 선 모양(점선 패턴)도 시리즈마다 다르게 둔다.
const SERIES_COLORS = [
  "var(--color-accent)",
  "var(--color-success)",
  "var(--color-warning)",
  "var(--color-danger)",
  "var(--color-text-secondary)",
];
const SERIES_DASHES = ["", "6 3", "2 3", "8 3 2 3", "1 4"];
const FONT_PX = 12;
const DEFAULT_WIDTH = 400;

/** 그릴 폭(px) — 글자 크기가 화면 폭에 따라 줄지 않도록 viewBox 를 실제 폭에 맞춘다. */
function useWidth() {
  const ref = useRef<HTMLDivElement>(null);
  const [width, setWidth] = useState(DEFAULT_WIDTH);
  useEffect(() => {
    const el = ref.current;
    if (!el || typeof ResizeObserver === "undefined") { return undefined; }
    const update = () => {
      const next = Math.round(el.clientWidth);
      if (next > 0) { setWidth((prev) => (prev === next ? prev : next)); }
    };
    update();
    const observer = new ResizeObserver(update);
    observer.observe(el);
    return () => observer.disconnect();
  }, []);
  return { ref, width };
}

function Legend({ data }: { data: ChartData }) {
  if (data.series.length < 2 && !data.series[0]?.name) { return null; }
  return (
    <ul className="mt-2 flex flex-wrap gap-x-4 gap-y-1 text-xs text-fg-secondary" aria-hidden="true">
      {data.series.map((s, index) => (
        <li key={index} className="flex items-center gap-1.5">
          <svg width="20" height="8" className="shrink-0">
            {data.kind === "bar" ? (
              <rect x="4" y="0" width="12" height="8" fill={SERIES_COLORS[index]} />
            ) : (
              <line x1="0" y1="4" x2="20" y2="4" stroke={SERIES_COLORS[index]} strokeWidth="2" strokeDasharray={SERIES_DASHES[index]} />
            )}
          </svg>
          {s.name}
        </li>
      ))}
    </ul>
  );
}

/** 스크린 리더용 대체 표(시각적으로 숨김). 표는 내용 폭 아래로 줄지 않으므로 sr-only 는 감싼 div 에 둔다. */
function ChartTable({ data }: { data: ChartData }) {
  const t = useT(blockMessages);
  return (
    <div className="sr-only">
      <table>
        <caption>{data.title || t.chart}</caption>
        <thead>
          <tr>
            <th scope="col">{data.x.label}</th>
            {data.series.map((s, index) => (
              <th key={index} scope="col">{s.name || data.y.label}</th>
            ))}
          </tr>
        </thead>
        <tbody>
          {data.x.values.map((x, r) => (
            <tr key={r}>
              <th scope="row">{x}</th>
              {data.series.map((s, index) => (
                <td key={index}>{formatValue(s.values[r])}</td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

/** `@@chart` — 선·막대, linear/log, 시리즈 최대 5개 */
export function ChartBlock({ data }: { data: ChartData }) {
  const { ref, width } = useWidth();
  const height = width < 480 ? 220 : 260;
  const layout = layoutChart(data, width, height, FONT_PX);
  const { plot } = layout;
  const maxLabel = Math.max(...data.x.values.map((v) => v.length));
  const labelEvery = Math.max(1, Math.ceil((maxLabel * FONT_PX * 0.62 + 8) / layout.band));
  const muted = "var(--color-text-muted)";

  return (
    <figure className="my-4 rounded-md border border-line bg-canvas-elevated p-3">
      {data.title && <figcaption className="mb-2 text-sm font-semibold text-fg">{data.title}</figcaption>}
      <div ref={ref} className="w-full">
        <svg width={width} height={height} viewBox={`0 0 ${width} ${height}`} className="block h-auto max-w-full" aria-hidden="true" fontSize={FONT_PX}>
          {data.y.label && (
            <text x={0} y={FONT_PX} fill={muted}>
              {data.y.label}
              {data.y.scale === "log" ? " (log)" : ""}
            </text>
          )}
          {layout.y.ticks.map((t) => (
            <g key={t.value}>
              <line x1={plot.left} x2={plot.right} y1={t.pos} y2={t.pos} stroke="var(--color-border)" strokeWidth={1} />
              <text x={plot.left - 6} y={t.pos} dy="0.35em" textAnchor="end" fill={muted}>
                {formatTick(t.value)}
              </text>
            </g>
          ))}
          {data.x.values.map((x, index) =>
            index % labelEvery === 0 ? (
              <text key={index} x={layout.xPos[index]} y={plot.bottom + FONT_PX * 1.4} textAnchor="middle" fill={muted}>
                {x}
              </text>
            ) : null,
          )}
          {data.x.label && (
            <text x={(plot.left + plot.right) / 2} y={height - 2} textAnchor="middle" fill={muted}>
              {data.x.label}
            </text>
          )}
          {data.series.map((s, si) =>
            data.kind === "bar" ? (
              <g key={si} fill={SERIES_COLORS[si]}>
                {s.values.map((v, pi) => {
                  const r = barRect(si, data.series.length, pi, v, layout);
                  return <rect key={pi} x={r.x} y={r.y} width={r.width} height={r.height} />;
                })}
              </g>
            ) : (
              <g key={si} stroke={SERIES_COLORS[si]} fill="none">
                <path d={linePath(s.values, layout)} strokeWidth={2} strokeDasharray={SERIES_DASHES[si]} strokeLinejoin="round" />
                {s.values.map((v, pi) => (
                  <circle key={pi} cx={layout.xPos[pi]} cy={layout.y.toPx(v)} r={2.5} fill={SERIES_COLORS[si]} />
                ))}
              </g>
            ),
          )}
        </svg>
      </div>
      <Legend data={data} />
      <ChartTable data={data} />
      {data.note && <p className="mt-2 text-sm text-fg-secondary">{data.note}</p>}
    </figure>
  );
}
