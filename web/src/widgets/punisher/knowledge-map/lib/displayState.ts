import { mapNodeState, type MapNodeState } from "@/src/entities/punisher";
import type { GraphNode } from "@/src/shared/api/punisher";
import type { messages } from "../i18n";

/** 지도에 보이는 상태 — 노드 색 상태(MapNodeState)에 "요청됨"(공부할 주제로 요청한 개념)·"필요 없음"(추천에서 뺀 개념)을 더한다. */
export type MapDisplayState = MapNodeState | "requested" | "dismissed";

type StateMessages = Pick<
  (typeof messages)["ko"],
  "stateCompleted" | "stateInProgress" | "stateKnown" | "stateRecommended" | "stateRequested" | "stateDismissed" | "stateLocked" | "stateNotStarted"
>;

type StateSource = Pick<GraphNode, "state" | "locked" | "frontier"> & { requests?: GraphNode["requests"]; dismissed?: boolean };

/** 배움·읽는 중·이미 앎은 그대로, 그 밖이면 뺀 개념은 필요 없음 → 요청이 붙어 있으면 요청됨 → 추천 → 잠김 → 미학습. */
export function mapDisplayState(source: StateSource): MapDisplayState {
  const base = mapNodeState(source);
  if (base === "completed" || base === "known" || base === "in_progress") { return base; }
  if (source.dismissed) { return "dismissed"; }
  return (source.requests?.length ?? 0) > 0 ? "requested" : base;
}

/** 색·테두리·아이콘에 쓰는 상태 — 요청됨은 밑바탕 상태를 따른다. */
export function styleState(source: StateSource): MapNodeState {
  return mapNodeState(source);
}

export function stateLabel(t: StateMessages, state: MapDisplayState): string {
  switch (state) {
    case "completed": return t.stateCompleted;
    case "in_progress": return t.stateInProgress;
    case "known": return t.stateKnown;
    case "recommended": return t.stateRecommended;
    case "requested": return t.stateRequested;
    case "dismissed": return t.stateDismissed;
    case "locked": return t.stateLocked;
    default: return t.stateNotStarted;
  }
}
