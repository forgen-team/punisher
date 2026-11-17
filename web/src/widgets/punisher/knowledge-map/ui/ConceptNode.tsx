
import { memo } from "react";
import { MessageCircleQuestion } from "lucide-react";
import { Handle, Position, type Node, type NodeProps } from "@xyflow/react";
import { cn } from "@/src/shared/ui/kit";
import {
  MAP_STATE_LABELS,
  MAP_STATE_STYLES,
  MapStateIcon,
  type MapNodeState,
} from "@/src/entities/punisher";
import { NODE_HEIGHT, nodeWidth } from "../lib/elkLayout";
import { messages } from "../i18n";
import { useT } from "@/src/shared/i18n";

export interface ConceptNodeData extends Record<string, unknown> {
  name: string;
  depth: number;
  displayState: MapNodeState;
  unlocks: number;
  /** 대기 중인 학습 요청 수(RSS "30분 학습" 등) */
  requested: number;
  /** 커리큘럼 밖에서 요청으로 제안된 개념 */
  proposed: boolean;
  /** 이 개념에 대한 내 질문('이 질문에 답하기' 해설) 수 */
  inquiries: number;
  /** 숙달한 개념 — 상태 색은 그대로 두고 테두리 링만 더한다 */
  mastered: boolean;
  dimmed: boolean;
  highlighted: boolean;
  selected: boolean;
  onSelect: (id: string) => void;
}

export type ConceptFlowNode = Node<ConceptNodeData, "concept">;

function ConceptNodeImpl({ id, data }: NodeProps<ConceptFlowNode>) {
  const style = MAP_STATE_STYLES[data.displayState];
  const recommended = data.displayState === "recommended";
  const t = useT(messages);
  return (
    <>
      <Handle type="target" position={Position.Left} className="!size-1.5 !border-0 !bg-line-strong" />
      <button
        type="button"
        onClick={() => data.onSelect(id)}
        aria-pressed={data.selected}
        aria-label={`${data.name}, ${MAP_STATE_LABELS[data.displayState]}${recommended ? t.nodeUnlocks(data.unlocks) : ""}${data.requested > 0 ? t.nodeRequested : ""}${data.proposed ? t.nodeProposed : ""}${data.inquiries > 0 ? t.nodeInquiries(data.inquiries) : ""}${data.mastered ? t.nodeMastered : ""}`}
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
          data.mastered && "outline outline-2 outline-offset-[5px] outline-success",
        )}
      >
        <span className="flex w-full items-center gap-1 font-semibold">
          <MapStateIcon state={data.displayState} />
          <span className="truncate">{data.name}</span>
        </span>
        <span className="flex w-full items-center gap-1 text-xs text-fg-secondary">
          {data.requested > 0 && (
            <span className="shrink-0 rounded-sm bg-accent-solid px-1 font-semibold text-accent-on">{t.requestedBadge}</span>
          )}
          <span className="min-w-0 truncate">
            {data.proposed ? t.proposed : MAP_STATE_LABELS[data.displayState]}
            {recommended && data.unlocks > 0 && t.nodeUnlocksShort(data.unlocks)}
          </span>
          {data.inquiries > 0 && (
            <span aria-hidden="true" className="ml-auto inline-flex shrink-0 items-center gap-0.5 rounded-sm bg-accent-subtle px-1 font-medium text-accent-text">
              <MessageCircleQuestion className="size-3" />
              {data.inquiries}
            </span>
          )}
        </span>
      </button>
      <Handle type="source" position={Position.Right} className="!size-1.5 !border-0 !bg-line-strong" />
    </>
  );
}

export const ConceptNode = memo(ConceptNodeImpl);
