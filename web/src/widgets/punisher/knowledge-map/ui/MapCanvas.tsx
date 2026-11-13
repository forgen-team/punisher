
// Lazy chunk: @xyflow/react + elkjs are only loaded on the map route.
import "@xyflow/react/dist/style.css";
import { useEffect, useMemo, useState } from "react";
import { Background, Controls, MiniMap, ReactFlow, type Edge } from "@xyflow/react";
import { MAP_STATE_STYLES, mapNodeState, useResolvedTheme } from "@/src/entities/punisher";
import type { GraphEdge, GraphNode } from "@/src/shared/api/punisher";
import { NODE_HEIGHT, layoutGraph, nodeWidth, type LaidOutNode } from "../lib/elkLayout";
import { FIT_MIN_ZOOM, initialFitIds } from "../lib/initialFit";
import { ConceptNode, type ConceptFlowNode } from "./ConceptNode";

const nodeTypes = { concept: ConceptNode };

export interface MapCanvasProps {
  nodes: GraphNode[];
  edges: GraphEdge[];
  highlightIds: Set<string> | null;
  selectedId: string | null;
  /** 숙달 링을 그릴 개념(숙달한 개념 전체) */
  masteredIds: ReadonlySet<string>;
  onSelect: (id: string) => void;
}

export default function MapCanvas({ nodes, edges, highlightIds, selectedId, masteredIds, onSelect }: MapCanvasProps) {
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
        // 화면 밖이라 아직 그리지 않은 노드도 크기를 알아야 fitView·미니맵이 그 노드를 센다.
        initialWidth: nodeWidth(n.depth),
        initialHeight: NODE_HEIGHT,
        data: {
          name: n.name,
          depth: n.depth,
          displayState: mapNodeState(n),
          unlocks: n.unlocks,
          requested: n.requests?.length ?? 0,
          proposed: n.status === "proposed",
          inquiries: n.inquiries?.length ?? 0,
          mastered: masteredIds.has(n.id),
          dimmed: highlightIds !== null && !highlightIds.has(n.id),
          highlighted: highlightIds?.has(n.id) ?? false,
          selected: n.id === selectedId,
          onSelect,
        },
      };
    });
  }, [nodes, positions, highlightIds, selectedId, masteredIds, onSelect]);

  // 첫 화면: 고른 개념 → 진행 중 → 추천 개념 기준으로 맞춘다(좁힌 보기는 전체). 확대 하한은 FIT_MIN_ZOOM.
  const fitIds = useMemo(() => initialFitIds(nodes, selectedId), [nodes, selectedId]);

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
    return <p role="alert" className="p-4 text-sm text-danger">지도 배치를 계산하지 못했어요.</p>;
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
      // 첫 확대는 읽을 수 있는 하한(FIT_MIN_ZOOM) 위로. 직접 축소는 더 멀리까지 되고, 화면 밖 노드는 그리지 않는다.
      fitViewOptions={{ padding: 0.08, nodes: fitIds, minZoom: FIT_MIN_ZOOM, maxZoom: 1.1 }}
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
      <MiniMap
        pannable
        zoomable
        nodeColor={(node) => MAP_STATE_STYLES[(node as ConceptFlowNode).data.displayState].border}
        className="!hidden md:!block"
      />
    </ReactFlow>
  );
}
