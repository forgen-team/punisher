import { describe, expect, it } from "vitest";
import type { GraphNode } from "@/src/shared/api/punisher";
import { FIT_ALL_MAX_NODES, initialFitIds } from "../lib/initialFit";

const node = (id: string, patch: Partial<GraphNode> = {}) => ({ id, state: "not_started", locked: false, frontier: false, ...patch }) as unknown as GraphNode;
const many = (n: number) => Array.from({ length: n }, (_, index) => node(`c${index}`));

describe("initialFitIds", () => {
  it("고른 개념이 있으면 그 개념", () => {
    expect(initialFitIds([node("a"), node("b")], "b")).toEqual([{ id: "b" }]);
  });

  it("좁힌 보기(노드가 적음)는 전체를 맞춘다", () => {
    expect(initialFitIds([node("a", { state: "in_progress" }), node("b")], null)).toBeUndefined();
  });

  it("전체 지도는 진행 중 → 추천 개념 기준, 둘 다 없으면 전체", () => {
    const base = many(FIT_ALL_MAX_NODES + 5);
    expect(initialFitIds([...base, node("p", { state: "in_progress" }), node("r", { frontier: true })], null)).toEqual([{ id: "p" }]);
    expect(initialFitIds([...base, node("r", { frontier: true })], null)).toEqual([{ id: "r" }]);
    expect(initialFitIds(base, null)).toBeUndefined();
  });

  it("지도에 없는 고른 개념은 무시한다", () => {
    expect(initialFitIds([node("a")], "zz")).toBeUndefined();
  });
});
