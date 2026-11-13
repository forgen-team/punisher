/** 경량 SVG 차트의 스케일·눈금·좌표 계산(라이브러리 없이). 화면 좌표는 위가 0 이다. */
import type { ChartData, ChartScaleKind } from "./blockData";

export interface Tick {
  value: number;
  pos: number;
}

export interface Scale {
  /** 값 → 화면 좌표 */
  toPx: (value: number) => number;
  ticks: Tick[];
  /** 막대의 바닥 값(linear 0 · log 최솟값 눈금) */
  baseline: number;
}

function niceStep(span: number, count: number): number {
  const raw = span / Math.max(1, count);
  const magnitude = 10 ** Math.floor(Math.log10(raw));
  const residual = raw / magnitude;
  const nice = [10, 5, 2].find((step, index) => residual >= [7.5, 3.5, 1.5][index]) ?? 1;
  return nice * magnitude;
}

/** 0 을 포함하는(막대 기준선) 보기 좋은 눈금의 선형 스케일 */
export function linearScale(values: number[], range: [number, number], count = 4): Scale {
  let min = Math.min(0, ...values);
  let max = Math.max(0, ...values);
  if (min === max) { max = min + 1; }
  const step = niceStep(max - min, count);
  min = Math.floor(min / step) * step;
  max = Math.ceil(max / step) * step;
  const [r0, r1] = range;
  const map = (v: number) => r0 + ((v - min) / (max - min)) * (r1 - r0);
  const ticks: Tick[] = [];
  for (let v = min; v <= max + step / 2; v += step) {
    const value = Number(v.toPrecision(12));
    ticks.push({ value, pos: map(value) });
  }
  return { toPx: map, ticks, baseline: 0 };
}

/** 10 의 거듭제곱 눈금의 로그 스케일(값은 모두 양수여야 한다) */
export function logScale(values: number[], range: [number, number]): Scale {
  const lo = Math.floor(Math.log10(Math.min(...values)));
  let hi = Math.ceil(Math.log10(Math.max(...values)));
  if (lo === hi) { hi = lo + 1; }
  const [r0, r1] = range;
  const map = (v: number) => r0 + ((Math.log10(v) - lo) / (hi - lo)) * (r1 - r0);
  // 자릿수가 많으면 눈금을 건너뛰어 최대 6개로 둔다(맨 위 눈금은 항상 둔다).
  const every = Math.ceil((hi - lo + 1) / 6);
  const ticks: Tick[] = [];
  for (let exponent = lo; exponent <= hi; exponent += every) { ticks.push({ value: 10 ** exponent, pos: map(10 ** exponent) }); }
  if (ticks[ticks.length - 1].value !== 10 ** hi) { ticks.push({ value: 10 ** hi, pos: map(10 ** hi) }); }
  return { toPx: map, ticks, baseline: 10 ** lo };
}

export function makeScale(kind: ChartScaleKind, values: number[], range: [number, number]): Scale {
  return kind === "log" ? logScale(values, range) : linearScale(values, range);
}

const COMPACT = new Intl.NumberFormat("ko-KR", { notation: "compact", maximumFractionDigits: 1 });
const PLAIN = new Intl.NumberFormat("ko-KR", { maximumFractionDigits: 3 });

export function formatTick(value: number): string {
  return Math.abs(value) >= 10_000 ? COMPACT.format(value) : PLAIN.format(value);
}

export function formatValue(value: number): string {
  return PLAIN.format(value);
}

export interface ChartLayout {
  width: number;
  height: number;
  plot: { left: number; right: number; top: number; bottom: number };
  /** x 칸 가운데 좌표(범주형, 같은 간격) */
  xPos: number[];
  /** 한 칸의 폭 */
  band: number;
  y: Scale;
}

/** 범주형 x(같은 간격)와 y 스케일로 그릴 자리를 정한다. 왼쪽 여백은 가장 긴 눈금 글자에 맞춘다. */
export function layoutChart(data: ChartData, width: number, height: number, fontPx = 12): ChartLayout {
  const all = data.series.flatMap((s) => s.values);
  const top = data.y.label ? fontPx * 2 : fontPx;
  const bottom = height - fontPx * (data.x.label ? 3.2 : 2);
  const probe = makeScale(data.y.scale, all, [bottom, top]);
  const labelWidth = Math.max(...probe.ticks.map((t) => formatTick(t.value).length)) * fontPx * 0.62;
  const left = Math.ceil(labelWidth + fontPx * 0.8);
  const right = width - fontPx * 0.5;
  const band = (right - left) / data.x.values.length;
  const xPos = data.x.values.map((_, index) => left + band * (index + 0.5));
  return { width, height, plot: { left, right, top, bottom }, xPos, band, y: probe };
}

/** 선 시리즈의 SVG path(d) */
export function linePath(values: number[], layout: ChartLayout): string {
  return values
    .map((v, index) => `${index === 0 ? "M" : "L"}${layout.xPos[index].toFixed(1)},${layout.y.toPx(v).toFixed(1)}`)
    .join("");
}

export interface BarRect {
  x: number;
  y: number;
  width: number;
  height: number;
}

/** 묶은 막대 — 칸 폭의 80% 를 시리즈 수로 나눈다. 음수 값은 기준선 아래로 그린다. */
export function barRect(seriesIndex: number, seriesCount: number, pointIndex: number, value: number, layout: ChartLayout): BarRect {
  const groupWidth = layout.band * 0.8;
  const width = groupWidth / seriesCount;
  const x = layout.xPos[pointIndex] - groupWidth / 2 + width * seriesIndex;
  const base = layout.y.toPx(layout.y.baseline);
  const top = layout.y.toPx(value);
  return { x, y: Math.min(base, top), width: Math.max(1, width - 1), height: Math.abs(base - top) };
}
