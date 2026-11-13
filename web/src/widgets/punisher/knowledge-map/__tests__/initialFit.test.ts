import { describe, expect, it } from "vitest";
import type { GraphNode } from "@/src/shared/api/punisher";
import { FIT_ALL_MAX_NODES, FIT_MIN_ZOOM, FIT_PADDING, fitCandidates, fitsAtMinZoom } from "../lib/initialFit";

const node = (id: string, patch: Partial<GraphNode> = {}) => ({ id, state: "not_started", locked: false, frontier: false, ...patch }) as unknown as GraphNode;
const many = (n: number) => Array.from({ length: n }, (_, index) => node(`c${index}`));

describe("fitCandidates", () => {
  it("고른 개념이 있으면 그 개념", () => {
    expect(fitCandidates([node("a"), node("b")], "b")).toEqual([[{ id: "b" }]]);
  });

  it("좁힌 보기: 전체 → 진행·추천 노드 → 최근 진행 노드 하나", () => {
    const nodes = [node("a", { state: "in_progress" }), node("b"), node("p", { state: "in_progress" }), node("r", { frontier: true })];
    expect(fitCandidates(nodes, null, ["p"])).toEqual([undefined, [{ id: "a" }, { id: "p" }, { id: "r" }], [{ id: "p" }]]);
    expect(fitCandidates([node("a"), node("b")], null)).toEqual([undefined]);
  });

  it("전체 지도: 진행 중이 여럿이면 최근에 본 1개, 없으면 추천 노드 → 첫 추천, 둘 다 없으면 전체", () => {
    const base = many(FIT_ALL_MAX_NODES + 5);
    const p1 = node("p1", { state: "in_progress" });
    const p2 = node("p2", { state: "in_progress" });
    expect(fitCandidates([...base, p1, p2, node("r", { frontier: true })], null, ["x", "p2", "p1"])).toEqual([[{ id: "p2" }]]);
    expect(fitCandidates([...base, p1, p2], null)).toEqual([[{ id: "p1" }]]);
    expect(fitCandidates([...base, node("r", { frontier: true }), node("s", { frontier: true })], null)).toEqual([[{ id: "r" }, { id: "s" }], [{ id: "r" }]]);
    expect(fitCandidates(base, null)).toEqual([undefined]);
  });

  it("지도에 없는 고른 개념은 무시한다", () => {
    expect(fitCandidates([node("a")], "zz")).toEqual([undefined]);
  });
});

describe("fitsAtMinZoom", () => {
  const box = (x: number, y: number) => ({ x, y, width: 150, height: 60 });

  it("하한 배율로 다 들어가면 true, 멀리 떨어진 두 노드는 false", () => {
    expect(fitsAtMinZoom([box(0, 0), box(300, 100)], 800, 500)).toBe(true);
    expect(fitsAtMinZoom([box(0, 0), box(2000, 0)], 800, 500)).toBe(false);
  });

  it("경계: 필요한 배율이 하한 바로 위면 들어가고 아래면 안 들어간다", () => {
    const width = 150 * FIT_MIN_ZOOM * (1 + FIT_PADDING);
    expect(fitsAtMinZoom([box(0, 0)], width + 0.01, 1000)).toBe(true);
    expect(fitsAtMinZoom([box(0, 0)], width - 1, 1000)).toBe(false);
  });

  it("상자나 화면 크기가 없으면 false", () => {
    expect(fitsAtMinZoom([], 800, 500)).toBe(false);
    expect(fitsAtMinZoom([box(0, 0)], 0, 0)).toBe(false);
  });
});
