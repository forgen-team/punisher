import { List, ListRow, Section } from "@forgen-team/ui";
import { areaLabel } from "@/src/entities/punisher";
import { PUNISHER_AREAS, type AreaCoverage, type GraphNode } from "@/src/shared/api/punisher";
import { useT } from "@/src/shared/i18n";
import { messages } from "../i18n";
import { mapDisplayState, stateLabel } from "../lib/displayState";
import { coveragePercent } from "./MapControls";
import { MapStateSwatch } from "./MapPanels";

interface MapConceptListProps {
  /** 지도에 보이는 개념(영역·목표·챕터·0단계 숨김을 이미 거른 것 — 그래프와 같은 목록) */
  nodes: GraphNode[];
  /** 영역 머리에 진행 %를 붙이기 위한 영역 요약(없으면 % 없음) */
  coverage?: AreaCoverage[];
  selectedId: string | null;
  onSelect: (id: string) => void;
}

/** 영역별로 묶는다(영역 순서는 PUNISHER_AREAS, 그 밖은 뒤). 영역 안은 깊이 순, 같은 깊이는 지도 데이터 순서 그대로. */
export function groupByArea(nodes: GraphNode[]): Array<{ area: string; nodes: GraphNode[] }> {
  const groups = new Map<string, GraphNode[]>();
  for (const node of nodes) {
    const list = groups.get(node.area);
    if (list) { list.push(node); } else { groups.set(node.area, [node]); }
  }
  const known: string[] = [...PUNISHER_AREAS];
  const rank = (area: string) => { const i = known.indexOf(area); return i === -1 ? known.length : i; };
  return [...groups.entries()]
    .sort(([a], [b]) => rank(a) - rank(b))
    .map(([area, list]) => ({ area, nodes: [...list].sort((a, b) => a.depth - b.depth) }));
}

/** 지도 목록 보기(폰 기본) — 영역별 묶음, 행 = 개념 이름 + 상태. 누르면 개념 상세가 열린다. */
export function MapConceptList({ nodes, coverage = [], selectedId, onSelect }: MapConceptListProps) {
  const t = useT(messages);
  return (
    <div className="flex flex-col gap-6">
      {groupByArea(nodes).map((group) => {
        const row = coverage.find((c) => c.area === group.area);
        return (
          <Section
            key={group.area}
            title={areaLabel(group.area)}
            action={row ? <span className="text-xs text-fg-muted">{t.areaPercent(coveragePercent(row))}</span> : undefined}
          >
            <List>
              {group.nodes.map((node) => {
                const state = mapDisplayState(node);
                const meta = node.status === "proposed" && state !== "dismissed" ? t.proposed : stateLabel(t, state);
                return (
                  <ListRow
                    key={node.id}
                    onClick={() => onSelect(node.id)}
                    selected={node.id === selectedId}
                    leading={<MapStateSwatch state={state} />}
                    title={node.name}
                    meta={meta}
                    badge={state === "recommended" ? { label: t.stateRecommended, tone: "accent" } : state === "requested" ? { label: t.stateRequested } : undefined}
                  />
                );
              })}
            </List>
          </Section>
        );
      })}
    </div>
  );
}
