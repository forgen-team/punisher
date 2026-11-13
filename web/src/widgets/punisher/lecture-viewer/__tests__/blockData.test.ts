import { describe, expect, it } from "vitest";
import { blockJson, parseChart, parseDiff, parseTrace } from "../lib/blockData";

const chart = {
  kind: "line",
  title: "n 에 따른 연산 수",
  x: { label: "n", values: [1, 10, 100] },
  y: { label: "연산 수", scale: "log" },
  series: [{ name: "n²", values: [1, 100, 10000] }],
};

describe("blockJson", () => {
  it("reads an object, a JSON string, or falls back to text", () => {
    expect(blockJson({ data: { a: 1 } })).toEqual({ a: 1 });
    expect(blockJson({ data: '{"a":2}' })).toEqual({ a: 2 });
    expect(blockJson({ text: '{"a":3}' })).toEqual({ a: 3 });
    expect(blockJson({ text: "not json" })).toBeUndefined();
  });
});

describe("parseTrace", () => {
  const ok = { caption: "합", columns: ["단계", "값"], rows: [["1", 2], ["2", "4"]] };
  it("accepts 2~6 columns, 2~16 rows and turns numbers into strings", () => {
    const parsed = parseTrace(ok);
    expect(parsed.ok && parsed.data.rows).toEqual([["1", "2"], ["2", "4"]]);
  });
  it.each([
    ["row length ≠ columns", { ...ok, rows: [["1"], ["2", "4"]] }],
    ["one column", { ...ok, columns: ["a"], rows: [["1"], ["2"]] }],
    ["one row", { ...ok, rows: [["1", "2"]] }],
    ["17 rows", { ...ok, rows: Array.from({ length: 17 }, () => ["1", "2"]) }],
    ["object cell", { ...ok, rows: [["1", {}], ["2", "4"]] }],
  ])("rejects %s", (_, raw) => {
    expect(parseTrace(raw).ok).toBe(false);
  });
});

describe("parseChart", () => {
  it("accepts a valid log chart", () => {
    const parsed = parseChart(chart);
    expect(parsed.ok).toBe(true);
    expect(parsed.ok && parsed.data.x.values).toEqual(["1", "10", "100"]);
  });
  it("defaults the scale to linear", () => {
    const parsed = parseChart({ ...chart, y: { label: "" } });
    expect(parsed.ok && parsed.data.y.scale).toBe("linear");
  });
  it.each([
    ["unknown kind", { ...chart, kind: "pie" }],
    ["unknown scale", { ...chart, y: { scale: "sqrt" } }],
    ["length mismatch", { ...chart, series: [{ name: "a", values: [1, 2] }] }],
    ["non-finite", { ...chart, y: { scale: "linear" }, series: [{ name: "a", values: [1, Number.NaN, 3] }] }],
    ["non-positive on log", { ...chart, series: [{ name: "a", values: [0, 1, 2] }] }],
    ["6 series", { ...chart, series: Array.from({ length: 6 }, () => ({ name: "a", values: [1, 2, 3] })) }],
    ["one point", { ...chart, x: { values: [1] }, series: [{ name: "a", values: [1] }] }],
    ["no series", { ...chart, series: [] }],
  ])("rejects %s", (_, raw) => {
    expect(parseChart(raw).ok).toBe(false);
  });
});

describe("parseDiff", () => {
  it("accepts different before/after", () => {
    expect(parseDiff({ before: "a", after: "b", note: "n" })).toEqual({ ok: true, data: { before: "a", after: "b", note: "n" } });
  });
  it("rejects identical text and more than 60 lines", () => {
    expect(parseDiff({ before: "a", after: "a" }).ok).toBe(false);
    expect(parseDiff({ before: Array(61).fill("x").join("\n"), after: "b" }).ok).toBe(false);
    expect(parseDiff({ before: 1, after: "b" }).ok).toBe(false);
  });
});
