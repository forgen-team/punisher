import { List, ListRow, Section } from "@forgen-team/ui";
import { MAP_STATE_LABELS, areaLabel, mapNodeState } from "@/src/entities/punisher";
import type { GraphNode } from "@/src/shared/api/punisher";
import { useT } from "@/src/shared/i18n";
import { messages } from "../i18n";
import { MapStateSwatch } from "./MapPanels";

interface MapConceptListProps {
  /** 지도에 보이는 개념(영역·목표·챕터·0단계 숨김을 이미 거른 것 — 그래프와 같은 목록) */
  nodes: GraphNode[];
  selectedId: string | null;
  /** 전체 영역을 볼 때만 행에 영역 이름을 붙인다 */
  showArea: boolean;
  onSelect: (id: string) => void;
}

/** 깊이 층(0단계 기초 → 깊이 3 심화)별로 묶는다. 층 안 순서는 지도 데이터 순서 그대로, 빈 층은 뺀다. */
export function groupByDepth(nodes: GraphNode[]): Array<{ depth: number; nodes: GraphNode[] }> {
  const layers = new Map<number, GraphNode[]>();
  for (const node of nodes) {
    const layer = layers.get(node.depth);
    if (layer) { layer.push(node); } else { layers.set(node.depth, [node]); }
  }
  return [...layers.entries()].sort(([a], [b]) => a - b).map(([depth, list]) => ({ depth, nodes: list }));
}

/** 지도 목록 보기(휴대폰 기본) — 노드와 같은 상태 견본·이름, 누르면 같은 개념 상세 시트가 열린다. */
export function MapConceptList({ nodes, selectedId, showArea, onSelect }: MapConceptListProps) {
  const t = useT(messages);
  return (
    <div className="flex flex-col gap-6">
      {groupByDepth(nodes).map((layer) => (
        <Section key={layer.depth} title={t.layerTitle(layer.depth)} action={<span className="text-xs text-fg-muted">{t.layerCount(layer.nodes.length)}</span>}>
          <List>
            {layer.nodes.map((node) => {
              const state = mapNodeState(node);
              const meta = [
                node.dismissed ? t.dismissedBadge : "",
                node.status === "proposed" ? t.proposed : MAP_STATE_LABELS[state],
                showArea ? areaLabel(node.area) : "",
              ].filter(Boolean).join(" · ");
              return (
                <ListRow
                  key={node.id}
                  onClick={() => onSelect(node.id)}
                  selected={node.id === selectedId}
                  leading={<MapStateSwatch state={state} />}
                  title={node.name}
                  meta={meta}
                />
              );
            })}
          </List>
        </Section>
      ))}
    </div>
  );
}
