import type { GraphNode } from "@/src/shared/api/punisher";
import type { MapNodeState } from "../ui/constants";

/** Display state of a knowledge-map node (design §3.6). */
export function mapNodeState(node: Pick<GraphNode, "state" | "locked" | "frontier">): MapNodeState {
  if (node.state === "completed") { return "completed"; }
  if (node.state === "known") { return "known"; }
  if (node.state === "in_progress") { return "in_progress"; }
  if (node.frontier) { return "recommended"; }
  if (node.locked) { return "locked"; }
  return "not_started";
}

/**
 * 상태별 노드 스타일(React Flow 인라인 style용) — W5 시맨틱 토큰 변수만 사용.
 * 강조색은 "추천(다음 학습)"에만. 색 외에 테두리 모양(solid/double/dashed/dotted)과 아이콘으로도 구분된다.
 */
export const MAP_STATE_STYLES: Record<MapNodeState, { border: string; background: string; borderStyle: string }> = {
  completed: { border: "var(--color-success)", background: "var(--color-success-subtle)", borderStyle: "solid" },
  known: { border: "var(--color-success)", background: "var(--color-bg-elevated)", borderStyle: "double" },
  "in_progress": { border: "var(--color-warning)", background: "var(--color-warning-subtle)", borderStyle: "dashed" },
  recommended: { border: "var(--color-accent)", background: "var(--color-accent-subtle)", borderStyle: "solid" },
  "not_started": { border: "var(--color-border-strong)", background: "var(--color-bg-elevated)", borderStyle: "solid" },
  locked: { border: "var(--color-border)", background: "var(--color-bg-subtle)", borderStyle: "dotted" },
};
