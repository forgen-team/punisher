/**
 * 구조 블록(trace·chart·diff) 데이터 검증·변환 — 설계 v2.2 §6 표의 규칙 그대로.
 * 서버도 검증하지만, 깨진 데이터가 와도 화면 전체가 멈추지 않도록 블록 단위로 다시 확인한다.
 */
import type { LectureBlock } from "@/src/shared/api/punisher";

export type Parsed<T> = { ok: true; data: T } | { ok: false; reason: string };

export interface TraceData {
  caption: string;
  columns: string[];
  rows: string[][];
  note: string;
}

export type ChartScaleKind = "linear" | "log";

export interface ChartSeries {
  name: string;
  values: number[];
}

export interface ChartData {
  kind: "line" | "bar";
  title: string;
  x: { label: string; values: string[] };
  y: { label: string; scale: ChartScaleKind };
  series: ChartSeries[];
  note: string;
}

export interface DiffData {
  before: string;
  after: string;
  note: string;
}

type Obj = Record<string, unknown>;

function isObj(value: unknown): value is Obj {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function str(value: unknown): string {
  return typeof value === "string" ? value : "";
}

function cell(value: unknown): string | null {
  if (typeof value === "string") { return value; }
  if (typeof value === "number" && Number.isFinite(value)) { return String(value); }
  return null;
}

function fail<T>(reason: string): Parsed<T> {
  return { ok: false, reason };
}

/** block.data(객체 또는 JSON 문자열)를 읽는다. data 가 없으면 본문(text)을 JSON 으로 본다. */
export function blockJson(block: Pick<LectureBlock, "data" | "text">): unknown {
  const raw = block.data ?? block.text;
  if (typeof raw !== "string") { return raw; }
  try {
    return JSON.parse(raw) as unknown;
  } catch {
    return undefined;
  }
}

export function parseTrace(raw: unknown): Parsed<TraceData> {
  if (!isObj(raw) || !Array.isArray(raw.columns) || !Array.isArray(raw.rows)) { return fail("형식"); }
  const columns = raw.columns.map((value) => cell(value));
  if (columns.length < 2 || columns.length > 6 || columns.some((c) => c === null)) { return fail("열 수"); }
  if (raw.rows.length < 2 || raw.rows.length > 16) { return fail("행 수"); }
  const rows: string[][] = [];
  for (const row of raw.rows) {
    if (!Array.isArray(row) || row.length !== columns.length) { return fail("행 길이"); }
    const cells = row.map((value) => cell(value));
    if (cells.some((c) => c === null)) { return fail("칸 값"); }
    rows.push(cells as string[]);
  }
  return { ok: true, data: { caption: str(raw.caption), columns: columns as string[], rows, note: str(raw.note) } };
}

export function parseChart(raw: unknown): Parsed<ChartData> {
  if (!isObj(raw) || !isObj(raw.x) || !isObj(raw.y) || !Array.isArray(raw.series)) { return fail("형식"); }
  if (raw.kind !== "bar" && raw.kind !== "line") { return fail("종류"); }
  const kind: ChartData["kind"] = raw.kind;
  const scale: ChartScaleKind = raw.y.scale === "log" ? "log" : "linear";
  if (raw.y.scale !== undefined && raw.y.scale !== "log" && raw.y.scale !== "linear") { return fail("스케일"); }
  const xValues = Array.isArray(raw.x.values) ? raw.x.values.map((value) => cell(value)) : [];
  if (xValues.length < 2 || xValues.length > 20 || xValues.some((v) => v === null)) { return fail("점 수"); }
  if (raw.series.length < 1 || raw.series.length > 5) { return fail("시리즈 수"); }
  const series: ChartSeries[] = [];
  for (const s of raw.series) {
    if (!isObj(s) || !Array.isArray(s.values) || s.values.length !== xValues.length) { return fail("길이"); }
    const values = s.values as unknown[];
    if (!values.every((v): v is number => typeof v === "number" && Number.isFinite(v))) { return fail("수 아님"); }
    if (scale === "log" && values.some((v) => v <= 0)) { return fail("log 양수"); }
    series.push({ name: str(s.name), values });
  }
  return {
    ok: true,
    data: {
      kind,
      title: str(raw.title),
      x: { label: str(raw.x.label), values: xValues as string[] },
      y: { label: str(raw.y.label), scale },
      series,
      note: str(raw.note),
    },
  };
}

const MAX_DIFF_LINES = 60;

function lineCount(text: string): number {
  return text.split("\n").length;
}

export function parseDiff(raw: unknown): Parsed<DiffData> {
  if (!isObj(raw) || typeof raw.before !== "string" || typeof raw.after !== "string") { return fail("형식"); }
  if (raw.before === raw.after) { return fail("같음"); }
  if (lineCount(raw.before) > MAX_DIFF_LINES || lineCount(raw.after) > MAX_DIFF_LINES) { return fail("줄 수"); }
  return { ok: true, data: { before: raw.before, after: raw.after, note: str(raw.note) } };
}
