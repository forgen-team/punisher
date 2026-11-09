// Loaded only inside the lazily imported MapCanvas chunk.
import ELK from "elkjs/lib/elk.bundled.js";
import type { GraphEdge, GraphNode } from "@/src/shared/api/punisher";

export const NODE_HEIGHT = 60;

export function nodeWidth(depth: number): number {
  return 150 + Math.max(0, depth - 1) * 20;
}

const elk = new ELK();

export interface LaidOutNode {
  id: string;
  x: number;
  y: number;
}

/** Layered (Sugiyama) layout, prerequisites flow left → right. */
export async function layoutGraph(nodes: GraphNode[], edges: GraphEdge[]): Promise<Map<string, LaidOutNode>> {
  const result = await elk.layout({
    id: "root",
    layoutOptions: {
      "elk.algorithm": "layered",
      "elk.direction": "RIGHT",
      "elk.layered.spacing.nodeNodeBetweenLayers": "70",
      "elk.spacing.nodeNode": "28",
      "elk.layered.nodePlacement.strategy": "BRANDES_KOEPF",
    },
    children: nodes.map((n) => ({ id: n.id, width: nodeWidth(n.depth), height: NODE_HEIGHT })),
    edges: edges.map((edge, index) => ({ id: `e${index}`, sources: [edge.from], targets: [edge.to] })),
  });
  const out = new Map<string, LaidOutNode>();
  for (const child of result.children ?? []) {
    out.set(child.id, { id: child.id, x: child.x ?? 0, y: child.y ?? 0 });
  }
  return out;
}
