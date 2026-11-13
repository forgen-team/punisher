import {
  CHECK_DISPUTE_WINDOW_MS,
  CHECK_MAX_ANSWER_CHARS,
  type CheckDetail,
  type CheckItem,
  type CheckOutcome,
  type CheckResultItem,
  type CheckTurn,
  type LectureSection,
  type ObjectiveMastery,
} from "@/src/shared/api/punisher";

/** 세션의 문항 순서(itemIds)대로 정렬한 문항. */
export function orderedItems(detail: CheckDetail): CheckItem[] {
  const order = new Map(detail.session.itemIds.map((id, index) => [id, index]));
  return [...detail.items].sort((a, b) => (order.get(a.id) ?? 0) - (order.get(b.id) ?? 0));
}

/** 지금 답할 문항 id — 세션이 active 일 때만(current 우선, 없으면 cursor 위치). */
export function currentItemId(detail: CheckDetail): string | null {
  if (detail.session.status !== "active") { return null; }
  return detail.current?.itemId ?? detail.session.itemIds[detail.session.cursor] ?? null;
}

/** 상단 진행 "문항 n/N" — 끝난 세션은 N/N. */
export function progressOf(detail: CheckDetail): { index: number; total: number } {
  const total = detail.session.itemIds.length;
  if (detail.session.status === "finished") { return { index: total, total }; }
  return { index: Math.min(detail.session.cursor + 1, total), total };
}

/** 그 문항의 턴(seq 순). 첫 물음(ask)은 문항 prompt 와 같아 문항 머리에서 보여 주므로 뺀다. */
export function itemTurns(turns: CheckTurn[], itemId: string): CheckTurn[] {
  return turns.filter((t) => t.itemId === itemId && t.move !== "ask").sort((a, b) => a.seq - b.seq);
}

/** 문항을 끝낸 판정 턴(outcome 이 붙은 마지막 턴, 실패·이의 제외) — 서버 progressOf 와 같은 기준. */
export function judgedTurn(turns: CheckTurn[], itemId: string): CheckTurn | null {
  let found: CheckTurn | null = null;
  for (const t of [...turns].sort((a, b) => a.seq - b.seq)) {
    if (t.itemId !== itemId || t.status === "failed" || t.move === "dispute") { continue; }
    if (t.outcome) { found = t; }
  }
  return found;
}

const DISPUTABLE: ReadonlySet<CheckOutcome> = new Set<CheckOutcome>(["aided", "failed", "held", "unaided_partial"]);

/**
 * '내 답이 맞아요' 노출 — 목표 문항(통합 문항 제외)이 끝났고, 결과가 이의 대상이며(공개 제외),
 * 그 문항에 이의를 낸 적이 없고, 세션이 열려 있거나 끝난 지 24시간 안. 실행 중이면 숨긴다.
 * 서버 judgedTurn·disputeOpen 과 같은 조건이라 409 를 미리 피한다.
 */
export function disputeTarget(detail: CheckDetail, item: CheckItem, now: number): CheckTurn | null {
  if (detail.pending || !item.done || item.objectiveId === "") { return null; }
  const judged = judgedTurn(detail.turns, item.id);
  if (!judged?.outcome || !DISPUTABLE.has(judged.outcome) || judged.verdict === "revealed") { return null; }
  const disputed = detail.turns.some((t) => t.itemId === item.id && t.role === "tutor" && t.move === "dispute" && t.status !== "failed");
  if (disputed) { return null; }
  const { status, finishedAt } = detail.session;
  if (status === "active") { return judged; }
  if ((status === "finished" || status === "abandoned") && finishedAt) {
    return now - new Date(finishedAt).getTime() < CHECK_DISPUTE_WINDOW_MS ? judged : null;
  }
  return null;
}

/** '다시 시도'를 붙일 실패 턴 — 마지막 턴이 실패한 튜터 턴일 때만(서버 RetryCheck 와 같은 기준). */
export function retryableTurnId(detail: CheckDetail): string | null {
  if (detail.pending || detail.turns.length === 0) { return null; }
  const last = [...detail.turns].sort((a, b) => a.seq - b.seq)[detail.turns.length - 1];
  if (last.role !== "tutor" || last.status !== "failed") { return null; }
  return detail.session.status === "active" || last.move === "dispute" ? last.id : null;
}

/** 하단 입력창 — active 세션에 현재 문항이 있을 때만 둔다. */
export function showComposer(detail: CheckDetail): boolean {
  return detail.session.status === "active" && detail.current !== null;
}

/** '그냥 알려줘' — 서버가 허용(current.canReveal)하고 실행 중이 아닐 때만. */
export function showReveal(detail: CheckDetail): boolean {
  return !detail.pending && Boolean(detail.current?.canReveal);
}

/** 보내기 — 답할 차례이고 실행 중이 아니며 1~최대 글자. */
export function canSend(detail: CheckDetail, text: string): boolean {
  const trimmed = text.trim();
  return !detail.pending && Boolean(detail.current?.canAnswer) && trimmed.length > 0 && trimmed.length <= CHECK_MAX_ANSWER_CHARS;
}

export interface ResultRow extends CheckResultItem {
  text: string;
  queued: boolean;
}

/** 결과 요약의 목표별 줄 — 통합 문항(objectiveId "")은 목표 레벨이 없으므로 뺀다. */
export function resultRows(detail: CheckDetail): ResultRow[] {
  const mastery = new Map<string, ObjectiveMastery>((detail.mastery ?? []).map((m) => [m.objectiveId, m]));
  return (detail.session.result?.items ?? [])
    .filter((r) => r.objectiveId !== "")
    .map((r) => {
      const m = mastery.get(r.objectiveId);
      return { ...r, text: m?.text || r.objectiveId, queued: r.queued ?? m?.queued ?? false };
    });
}

/** '다시 볼 곳' 링크 글자 — 앵커가 가리키는 강의 섹션 제목(섹션·블록 앵커 모두). */
export function anchorTitle(sections: LectureSection[] | undefined, anchor: string): string {
  const section = (sections ?? []).find((s) => s.anchor === anchor || s.blocks.some((b) => b.anchor === anchor));
  return section?.title ?? "";
}
