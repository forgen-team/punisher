import type { GraphNode } from "@/src/shared/api/punisher";
import { mapNodeState } from "@/src/entities/punisher";

/** 첫 화면 확대의 하한 — 이보다 작으면 노드 글자를 읽을 수 없다(노드 150×60 기준). */
export const FIT_MIN_ZOOM = 0.6;

/** 이 수 이하이면(영역·챕터·목표로 좁힌 보기) 내 위치 대신 전체를 맞춘다. */
export const FIT_ALL_MAX_NODES = 30;

/**
 * 첫 화면에 맞출 노드 — 고른 개념 → (좁힌 보기면 전체) → 진행 중인 개념 → 추천(다음 학습) 개념 → 전체(undefined).
 * 전체 영역(~170개)을 한 화면에 넣으면 노드가 점처럼 작아지므로 내가 있는 곳을 기준으로 맞춘다.
 */
export function initialFitIds(nodes: GraphNode[], selectedId: string | null): { id: string }[] | undefined {
  if (selectedId && nodes.some((n) => n.id === selectedId)) { return [{ id: selectedId }]; }
  if (nodes.length <= FIT_ALL_MAX_NODES) { return undefined; }
  const inProgress = nodes.filter((n) => mapNodeState(n) === "in_progress");
  if (inProgress.length > 0) { return inProgress.map((n) => ({ id: n.id })); }
  const recommended = nodes.filter((n) => mapNodeState(n) === "recommended");
  if (recommended.length > 0) { return recommended.map((n) => ({ id: n.id })); }
  return undefined;
}
