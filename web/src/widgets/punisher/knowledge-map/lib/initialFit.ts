import type { GraphNode } from "@/src/shared/api/punisher";
import { mapNodeState } from "@/src/entities/punisher";

/** 맞춤 확대의 하한 — 이보다 작으면 노드 글자를 읽기 어렵다(노드 150×60 기준). */
export const FIT_MIN_ZOOM = 0.85;
export const FIT_MAX_ZOOM = 1.1;
export const FIT_PADDING = 0.08;

/** 이 수 이하이면(영역·챕터·목표로 좁힌 보기) 먼저 전체를 맞춰 본다. */
export const FIT_ALL_MAX_NODES = 30;

/** 맞출 노드 묶음 — undefined 는 전체. */
export type FitIds = { id: string }[] | undefined;

const ids = (list: string[]): { id: string }[] => list.map((id) => ({ id }));

/**
 * 첫 화면에 맞출 후보(앞에서부터). 캔버스는 FIT_MIN_ZOOM 으로 다 들어가는 첫 후보를 쓰고,
 * 하나도 안 들어가면 마지막 후보(기준 노드 하나)를 가운데에 둔다.
 * - 고른 개념 → 그 개념
 * - 좁힌 보기 → 전체 → 진행·추천 노드 → 기준 노드
 * - 전체 지도 → 진행 중이면 최근 1개, 아니면 추천 노드 → 추천 첫 노드, 둘 다 없으면 전체
 * 기준 노드: 진행 중 가운데 최근에 본 것(`recentConceptIds` 순서, 없으면 첫 진행 중) → 첫 추천.
 */
export function fitCandidates(nodes: GraphNode[], selectedId: string | null, recentConceptIds: readonly string[] = []): FitIds[] {
  if (selectedId && nodes.some((n) => n.id === selectedId)) { return [[{ id: selectedId }]]; }
  const inProgress = nodes.filter((n) => mapNodeState(n) === "in_progress").map((n) => n.id);
  const recommended = nodes.filter((n) => mapNodeState(n) === "recommended").map((n) => n.id);
  const recent = recentConceptIds.find((id) => inProgress.includes(id)) ?? inProgress[0];
  const anchor = recent ?? recommended[0];
  if (nodes.length <= FIT_ALL_MAX_NODES) {
    const focus = [...inProgress, ...recommended];
    return [undefined, ...(focus.length > 1 ? [ids(focus)] : []), ...(anchor ? [ids([anchor])] : [])];
  }
  if (recent) { return [ids([recent])]; }
  if (recommended.length > 0) { return [ids(recommended), ids([recommended[0]])]; }
  return [undefined];
}

export interface FitBox {
  x: number;
  y: number;
  width: number;
  height: number;
}

/** 이 상자들을 FIT_MIN_ZOOM 이상으로 화면(width×height)에 다 넣을 수 있는가(React Flow fitView 와 같은 계산). */
export function fitsAtMinZoom(boxes: readonly FitBox[], width: number, height: number): boolean {
  if (boxes.length === 0 || width <= 0 || height <= 0) { return false; }
  const left = Math.min(...boxes.map((b) => b.x));
  const top = Math.min(...boxes.map((b) => b.y));
  const right = Math.max(...boxes.map((b) => b.x + b.width));
  const bottom = Math.max(...boxes.map((b) => b.y + b.height));
  const zoom = Math.min(width / (1 + FIT_PADDING) / (right - left), height / (1 + FIT_PADDING) / (bottom - top));
  return zoom >= FIT_MIN_ZOOM;
}
