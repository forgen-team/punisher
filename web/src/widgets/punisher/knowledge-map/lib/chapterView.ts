import type { GraphEdge, GraphNode } from "@/src/shared/api/punisher";

/** 챕터 보기: 그 챕터의 개념 + 그 개념들의 바로 앞 선행 개념(지도의 기존 개념과 어떻게 이어지는지). */
export function chapterView(nodes: GraphNode[], edges: GraphEdge[], chapterId: string): GraphNode[] {
  const inChapter = new Set(nodes.filter((n) => n.chapterId === chapterId).map((n) => n.id));
  const keep = new Set(inChapter);
  for (const edge of edges) {
    if (edge.kind === "prerequisite" && inChapter.has(edge.to)) { keep.add(edge.from); }
  }
  return nodes.filter((n) => keep.has(n.id));
}
