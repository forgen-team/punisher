import { FOUNDATIONS_AREA, type GraphNode } from "@/src/shared/api/punisher";

/** URL 값: ?foundations=shown 이면 기본 숨김인 0단계 기초를 펼친다. */
export const FOUNDATIONS_SHOWN = "shown";

/**
 * 0단계 기초 노드 거르기(learner-level ②): 서버가 내 수준에서 기본 숨김으로 표시한 노드(hiddenByDefault)는
 * 펼치기 전까지 빼고, 0단계 기초 영역을 직접 고른 경우에는 그대로 둔다. 수준을 바꾸면 서버 표시만 바뀐다.
 */
export function visibleFoundations(nodes: GraphNode[], expanded: boolean, areaFilter: string | null): GraphNode[] {
  if (expanded || areaFilter === FOUNDATIONS_AREA) { return nodes; }
  return nodes.filter((n) => !n.hiddenByDefault);
}

/** 접혀 있는(기본 숨김) 0단계 기초 노드 수 — 0이면 펼치기 줄을 띄우지 않는다. */
export function hiddenFoundationsCount(nodes: GraphNode[]): number {
  return nodes.filter((n) => n.hiddenByDefault).length;
}
