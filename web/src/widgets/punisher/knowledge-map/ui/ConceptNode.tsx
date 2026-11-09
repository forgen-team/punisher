
import { memo } from "react";
import { Handle, Position, type Node, type NodeProps } from "@xyflow/react";
import { cn } from "@/src/shared/ui/kit";
import {
  MAP_STATE_LABELS,
  MAP_STATE_STYLES,
  MapStateIcon,
  type MapNodeState,
} from "@/src/entities/punisher";
import { NODE_HEIGHT, nodeWidth } from "../lib/elkLayout";

export interface ConceptNodeData extends Record<string, unknown> {
  name: string;
  depth: number;
  displayState: MapNodeState;
  unlocks: number;
  /** 대기 중인 학습 요청 수(RSS "30분 학습" 등) */
  requested: number;
  /** 커리큘럼 밖에서 요청으로 제안된 개념 */
  proposed: boolean;
  dimmed: boolean;
  highlighted: boolean;
  selected: boolean;
  onSelect: (id: string) => void;
}

export type ConceptFlowNode = Node<ConceptNodeData, "concept">;

function ConceptNodeImpl({ id, data }: NodeProps<ConceptFlowNode>) {
  const style = MAP_STATE_STYLES[data.displayState];
  const recommended = data.displayState === "recommended";
  return (
    <>
      <Handle type="target" position={Position.Left} className="!size-1.5 !border-0 !bg-line-strong" />
      <button
        type="button"
        onClick={() => data.onSelect(id)}
        aria-pressed={data.selected}
        aria-label={`${data.name}, ${MAP_STATE_LABELS[data.displayState]}${recommended ? `, 열리는 개념 ${data.unlocks}개` : ""}${data.requested > 0 ? ", 학습 요청됨" : ""}${data.proposed ? ", 제안 개념" : ""}`}
        style={{
          width: nodeWidth(data.depth),
          height: NODE_HEIGHT,
          borderColor: style.border,
          background: style.background,
          borderStyle: style.borderStyle,
          opacity: data.dimmed ? 0.3 : 1,
        }}
        className={cn(
          "flex flex-col items-start justify-center rounded-md px-2.5 text-left text-xs text-fg",
          "focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus",
          recommended || data.highlighted ? "border-[3px] shadow-sm" : "border-2",
          data.selected && "ring-2 ring-focus ring-offset-2 ring-offset-canvas",
        )}
      >
        <span className="flex w-full items-center gap-1 font-semibold">
          <MapStateIcon state={data.displayState} />
          <span className="truncate">{data.name}</span>
        </span>
        <span className="flex w-full items-center gap-1 text-xs text-fg-secondary">
          {data.requested > 0 && (
            <span className="shrink-0 rounded-sm bg-accent-solid px-1 font-semibold text-accent-on">요청됨</span>
          )}
          <span className="truncate">
            {data.proposed ? "제안 개념" : MAP_STATE_LABELS[data.displayState]}
            {recommended && data.unlocks > 0 && ` · +${data.unlocks}개 열림`}
          </span>
        </span>
      </button>
      <Handle type="source" position={Position.Right} className="!size-1.5 !border-0 !bg-line-strong" />
    </>
  );
}

export const ConceptNode = memo(ConceptNodeImpl);
