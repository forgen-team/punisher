
// Lazy chunk: @xyflow/react + elkjs are only loaded on the map route.
import "@xyflow/react/dist/style.css";
import { useEffect, useMemo, useState } from "react";
import { Background, Controls, MiniMap, ReactFlow, type Edge } from "@xyflow/react";
import { mapNodeState, useResolvedTheme } from "@/src/entities/punisher";
import type { GraphEdge, GraphNode } from "@/src/shared/api/punisher";
import { layoutGraph, type LaidOutNode } from "../lib/elkLayout";
import { ConceptNode, type ConceptFlowNode } from "./ConceptNode";

const nodeTypes = { concept: ConceptNode };

export interface MapCanvasProps {
  nodes: GraphNode[];
  edges: GraphEdge[];
  highlightIds: Set<string> | null;
  selectedId: string | null;
  /** 숙달 링을 그릴 개념(개념 단위 숙달은 상세를 연 개념만 받는다) */
  masteredId: string | null;
  onSelect: (id: string) => void;
}

export default function MapCanvas({ nodes, edges, highlightIds, selectedId, masteredId, onSelect }: MapCanvasProps) {
  const theme = useResolvedTheme();
  const [positions, setPositions] = useState<Map<string, LaidOutNode> | null>(null);
  const [layoutError, setLayoutError] = useState(false);

  useEffect(() => {
    let cancelled = false;
    setPositions(null);
    layoutGraph(nodes, edges)
      .then((pos) => {
        if (!cancelled) { setPositions(pos); }
      })
      .catch((err: unknown) => {
        console.warn("punisher: map layout failed", err);
        if (!cancelled) { setLayoutError(true); }
      });
    return () => {
      cancelled = true;
    };
  }, [nodes, edges]);

  const flowNodes = useMemo<ConceptFlowNode[]>(() => {
    if (!positions) { return []; }
    return nodes.map((n) => {
      const pos = positions.get(n.id);
      return {
        id: n.id,
        type: "concept",
        position: { x: pos?.x ?? 0, y: pos?.y ?? 0 },
        data: {
          name: n.name,
          depth: n.depth,
          displayState: mapNodeState(n),
          unlocks: n.unlocks,
          requested: n.requests?.length ?? 0,
          proposed: n.status === "proposed",
          inquiries: n.inquiries?.length ?? 0,
          mastered: n.id === masteredId,
          dimmed: highlightIds !== null && !highlightIds.has(n.id),
          highlighted: highlightIds?.has(n.id) ?? false,
          selected: n.id === selectedId,
          onSelect,
        },
      };
    });
  }, [nodes, positions, highlightIds, selectedId, masteredId, onSelect]);

  // 선택한 개념이 있으면 그 주변으로, 없으면 전체가 보이게 맞춘다.
  const fitIds = useMemo(() => (selectedId && nodes.some((n) => n.id === selectedId) ? [{ id: selectedId }] : undefined), [nodes, selectedId]);

  const flowEdges = useMemo<Edge[]>(
    () =>
      edges.map((edge) => {
        const onPath = highlightIds !== null && highlightIds.has(edge.from) && highlightIds.has(edge.to);
        return {
          id: `${edge.from}->${edge.to}`,
          source: edge.from,
          target: edge.to,
          animated: onPath,
          style: {
            stroke: onPath ? "var(--color-accent)" : "var(--color-border-strong)",
            strokeWidth: onPath ? 2.5 : 1.2,
            opacity: highlightIds !== null && !onPath ? 0.25 : 1,
          },
        };
      }),
    [edges, highlightIds],
  );

  if (layoutError) {
    return <p role="alert" className="p-4 text-sm text-danger">지도 배치를 계산하지 못했습니다.</p>;
  }
  if (!positions) {
    return <div aria-hidden="true" className="h-full w-full animate-pulse bg-canvas-hover motion-reduce:animate-none" />;
  }

  return (
    <ReactFlow
      nodes={flowNodes}
      edges={flowEdges}
      nodeTypes={nodeTypes}
      colorMode={theme}
      fitView
      // 전체 영역(~170개)이 한 화면에 들어오도록 더 멀리 축소할 수 있게 하고, 화면 밖 노드는 그리지 않는다.
      fitViewOptions={{ padding: 0.08, nodes: fitIds, maxZoom: 1.1 }}
      minZoom={0.05}
      onlyRenderVisibleElements
      nodesDraggable={false}
      nodesConnectable={false}
      elementsSelectable={false}
      // React Flow가 노드 내부 버튼의 마우스 클릭을 팬/드래그 처리로 가져가므로 노드 클릭은 여기서 받는다.
      // (노드 내부 버튼은 키보드 Enter/Space 선택용으로 유지)
      onNodeClick={(_, node) => onSelect(node.id)}
      aria-label="지식 지도"
    >
      <Background />
      <Controls showInteractive={false} />
      <MiniMap pannable zoomable className="!hidden md:!block" />
    </ReactFlow>
  );
}
