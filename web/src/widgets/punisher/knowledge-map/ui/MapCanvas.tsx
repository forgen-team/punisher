
// Lazy chunk: @xyflow/react + elkjs are only loaded on the map route.
import { useEffect, useMemo, useRef, useState } from "react";
import { Background, Controls, MiniMap, Panel, ReactFlow, type Edge, type ReactFlowInstance } from "@xyflow/react";
import { Maximize2, Minimize2 } from "lucide-react";
import { Button } from "@forgen-team/ui";
import { MAP_STATE_STYLES, mapNodeState, useResolvedTheme } from "@/src/entities/punisher";
import type { GraphEdge, GraphNode } from "@/src/shared/api/punisher";
import { NODE_HEIGHT, layoutGraph, nodeWidth, type LaidOutNode } from "../lib/elkLayout";
import { FIT_MAX_ZOOM, FIT_MIN_ZOOM, FIT_PADDING, fitCandidates, fitsAtMinZoom, type FitBox } from "../lib/initialFit";
import { ConceptNode, type ConceptFlowNode } from "./ConceptNode";
import { messages } from "../i18n";
import { useT } from "@/src/shared/i18n";

const nodeTypes = { concept: ConceptNode };

export interface MapCanvasProps {
  nodes: GraphNode[];
  edges: GraphEdge[];
  highlightIds: Set<string> | null;
  selectedId: string | null;
  /** 숙달 링을 그릴 개념(숙달한 개념 전체) */
  masteredIds: ReadonlySet<string>;
  /** 최근에 본 강의의 개념(최근 순) — 진행 중이 여럿일 때 첫 화면 기준 */
  recentConceptIds: readonly string[];
  onSelect: (id: string) => void;
}

function nodeBox(node: GraphNode, pos: LaidOutNode | undefined): FitBox {
  return { x: pos?.x ?? 0, y: pos?.y ?? 0, width: nodeWidth(node.depth), height: NODE_HEIGHT };
}

export default function MapCanvas({ nodes, edges, highlightIds, selectedId, masteredIds, recentConceptIds, onSelect }: MapCanvasProps) {
  const t = useT(messages);
  const theme = useResolvedTheme();
  const [positions, setPositions] = useState<Map<string, LaidOutNode> | null>(null);
  const [layoutError, setLayoutError] = useState(false);
  const wrapRef = useRef<HTMLDivElement>(null);
  const [size, setSize] = useState({ width: 0, height: 0 });
  const [flow, setFlow] = useState<ReactFlowInstance<ConceptFlowNode, Edge> | null>(null);
  const centeredFor = useRef(selectedId);

  useEffect(() => {
    let cancelled = false;
    setPositions(null);
    layoutGraph(nodes, edges)
      .then((pos) => {
        if (cancelled) { return; }
        const el = wrapRef.current;
        setSize({ width: el?.clientWidth ?? 0, height: el?.clientHeight ?? 0 });
        setPositions(pos);
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
          dismissed: Boolean(n.dismissed),
          dimmed: highlightIds !== null && !highlightIds.has(n.id),
          highlighted: highlightIds?.has(n.id) ?? false,
          selected: n.id === selectedId,
          onSelect,
        },
      };
    });
  }, [nodes, positions, highlightIds, selectedId, masteredIds, onSelect]);

  // 첫 화면: 후보(고른 개념 / 좁힌 보기 전체 → 진행·추천 / 최근 진행 1개 …) 가운데 FIT_MIN_ZOOM 으로 다 들어가는 첫 묶음,
  // 하나도 안 들어가면 기준 노드 하나를 가운데에 둔다. fitView 는 캔버스가 처음 그려질 때 한 번만 쓰인다.
  const fitIds = useMemo(() => {
    if (!positions) { return undefined; }
    const byId = new Map(nodes.map((n) => [n.id, n]));
    const boxes = (ids: { id: string }[] | undefined) =>
      (ids ?? nodes).flatMap((ref) => {
        const node = byId.get(ref.id);
        return node ? [nodeBox(node, positions.get(node.id))] : [];
      });
    const candidates = fitCandidates(nodes, selectedId, recentConceptIds);
    return candidates.find((ids) => fitsAtMinZoom(boxes(ids), size.width, size.height)) ?? candidates[candidates.length - 1];
  }, [nodes, positions, selectedId, recentConceptIds, size]);

  // 캔버스 밖(오른쪽 '다음에 공부하면 좋은 것' 등)에서 고른 개념이 화면에 없으면 그 노드로 옮긴다(배율은 유지, 하한 FIT_MIN_ZOOM).
  useEffect(() => {
    if (centeredFor.current === selectedId) { return; }
    centeredFor.current = selectedId;
    const node = selectedId ? nodes.find((n) => n.id === selectedId) : undefined;
    const el = wrapRef.current;
    if (!flow || !positions || !node || !el) { return; }
    const box = nodeBox(node, positions.get(node.id));
    const { x, y, zoom } = flow.getViewport();
    const visible = box.x * zoom + x >= 0 && box.y * zoom + y >= 0 &&
      (box.x + box.width) * zoom + x <= el.clientWidth && (box.y + box.height) * zoom + y <= el.clientHeight;
    if (visible) { return; }
    const reduced = window.matchMedia("(prefers-reduced-motion: reduce)").matches;
    void flow.setCenter(box.x + box.width / 2, box.y + box.height / 2, { zoom: Math.max(zoom, FIT_MIN_ZOOM), duration: reduced ? 0 : 300 });
  }, [flow, nodes, positions, selectedId]);

  // 첫 화면은 진행·추천 묶음에 맞춘다 — 그때 화면 밖에 개념이 더 있으면 "일부만 보여요" 단서와 '전체 보기'를 둔다(P3·점검 27).
  const partial = fitIds !== undefined && fitIds.length < nodes.length;
  const [showingAll, setShowingAll] = useState(false);
  useEffect(() => setShowingAll(false), [nodes]);
  const reduced = typeof window !== "undefined" && window.matchMedia?.("(prefers-reduced-motion: reduce)").matches;
  function showAll() {
    if (!flow) { return; }
    void flow.fitView({ padding: FIT_PADDING, minZoom: 0.05, maxZoom: FIT_MAX_ZOOM, duration: reduced ? 0 : 300 });
    setShowingAll(true);
  }
  function showFocus() {
    if (!flow) { return; }
    void flow.fitView({ padding: FIT_PADDING, nodes: fitIds, minZoom: FIT_MIN_ZOOM, maxZoom: FIT_MAX_ZOOM, duration: reduced ? 0 : 300 });
    setShowingAll(false);
  }

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
    return <p role="alert" className="p-4 text-sm text-danger">{t.layoutError}</p>;
  }

  return (
    <div ref={wrapRef} className="h-full w-full">
      {positions ? (
        <ReactFlow
          nodes={flowNodes}
          edges={flowEdges}
          nodeTypes={nodeTypes}
          colorMode={theme}
          onInit={setFlow}
          fitView
          // 첫 확대는 읽을 수 있는 하한(FIT_MIN_ZOOM) 위로. 직접 축소는 더 멀리까지 되고, 화면 밖 노드는 그리지 않는다.
          fitViewOptions={{ padding: FIT_PADDING, nodes: fitIds, minZoom: FIT_MIN_ZOOM, maxZoom: FIT_MAX_ZOOM }}
          minZoom={0.05}
          onlyRenderVisibleElements
          nodesDraggable={false}
          nodesConnectable={false}
          elementsSelectable={false}
          // React Flow가 노드 내부 버튼의 마우스 클릭을 팬/드래그 처리로 가져가므로 노드 클릭은 여기서 받는다.
          // (노드 내부 버튼은 키보드 Enter/Space 선택용으로 유지)
          onNodeClick={(_, node) => onSelect(node.id)}
          aria-label={t.mapLabel}
        >
          <Background />
          {partial && (
            <Panel position="top-left" className="!m-2">
              <div className="flex items-center gap-2 rounded-md border border-line bg-canvas-elevated py-0.5 pl-2.5 pr-0.5 text-xs text-fg-secondary shadow-xs">
                {showingAll ? null : <span>{t.partialView(fitIds?.length ?? 0, nodes.length)}</span>}
                <Button variant="ghost" size="sm" onClick={showingAll ? showFocus : showAll}>
                  {showingAll ? <Minimize2 className="size-3.5" aria-hidden="true" /> : <Maximize2 className="size-3.5" aria-hidden="true" />}
                  {showingAll ? t.focusView : t.showAll}
                </Button>
              </div>
            </Panel>
          )}
          {/* 휴대폰(fold 미만): 확대·축소·맞춤 버튼을 터치 크기(44px)로, 아이콘도 조금 크게. */}
          <Controls
            showInteractive={false}
            className="max-fold:[&_button]:size-touch max-fold:[&_svg]:!max-h-4 max-fold:[&_svg]:!max-w-4"
          />
          {/* 작게(기본 200×150 → 144×96), 바탕·바깥 가림은 토큰 색 — 다크에서 밝은 덩어리로 보이지 않게. 지금 보는 범위는 액센트 테두리. */}
          <MiniMap
            pannable
            zoomable
            nodeColor={(node) => MAP_STATE_STYLES[(node as ConceptFlowNode).data.displayState].border}
            bgColor="var(--color-bg-elevated)"
            maskColor="var(--color-bg-translucent)"
            maskStrokeColor="var(--color-accent)"
            maskStrokeWidth={2}
            style={{ width: 144, height: 96 }}
            className="!hidden overflow-hidden rounded-md border border-line md:!block"
          />
        </ReactFlow>
      ) : (
        <div aria-hidden="true" className="h-full w-full animate-pulse bg-canvas-hover motion-reduce:animate-none" />
      )}
    </div>
  );
}
