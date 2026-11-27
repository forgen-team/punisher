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
  type Prediction,
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

/** 진행 "n/N" — 끝난 세션은 N/N. */
export function progressOf(detail: CheckDetail): { index: number; total: number } {
  const total = detail.session.itemIds.length;
  if (detail.session.status === "finished") { return { index: total, total }; }
  return { index: Math.min(detail.session.cursor + 1, total), total };
}

/** 이 문항이 세션에서 몇 번째인지(1부터)와 전체 — 끝난 문항을 보는 중에도 그 문항의 자리를 가리킨다. */
export function progressAt(detail: CheckDetail, itemId: string | null): { index: number; total: number } {
  const total = detail.session.itemIds.length;
  const at = itemId ? detail.session.itemIds.indexOf(itemId) : -1;
  return at < 0 ? progressOf(detail) : { index: at + 1, total };
}

/** 진행 막대 비율(0~1) — 보는 문항이 끝났으면 그 문항까지 찬 것으로 센다. */
export function progressRatio(index: number, total: number, itemDone: boolean): number {
  if (total === 0) { return 0; }
  return Math.min(1, (itemDone ? index : index - 1) / total);
}

/** 복습 세션인가(같은 check/[id] 화면을 쓴다). */
export function isReview(detail: CheckDetail): boolean {
  return detail.session.kind === "review";
}

/** 확인 결과의 '이 강의 한 장 요약' — 한 강의의 세션이라 첫 요약 하나(없으면 undefined). */
export function sessionLectureSummary(detail: CheckDetail): string | undefined {
  return detail.items.find((item) => item.lectureSummary)?.lectureSummary;
}

/** 진행 줄 "3/12 · 복습" — 문항이 없으면 제목만. */
export function stepMeta(progress: { index: number; total: number }, title: string): string {
  return progress.total === 0 ? title : `${progress.index}/${progress.total} · ${title}`;
}


/** 결과 목표 줄 중 가장 이른 다음 복습일(밀려 대기인 것·날짜 없는 것 제외). 없으면 null. */
export function earliestNextReview(items: readonly CheckResultItem[] | undefined): string | null {
  const dates = (items ?? []).filter((r) => !r.queued && r.nextReviewAt).map((r) => r.nextReviewAt as string).sort();
  return dates[0] ?? null;
}

/** 그만두기 확인 창에 쓸 개수 — 답한 문항(끝낸 것 + 지금 문항에 답을 보낸 것)과 남은 문항. */
export function stopCounts(detail: CheckDetail, answeredCurrent: boolean): { answered: number; remaining: number } {
  const total = detail.session.itemIds.length;
  const answered = Math.min(total, detail.items.filter((item) => item.done).length + (answeredCurrent ? 1 : 0));
  return { answered, remaining: total - answered };
}

export type SelfGradeStep = "answer" | "grade";

/**
 * 자기 채점 단계 — 답 쓰기(answer) → 모범답·기준을 본 뒤 맞음/부분/틀림 고르기(grade).
 * 자기 채점 세션의 현재 문항에서 실행 중이 아닐 때만, 서버 current 의 허용(canAnswer/canSelfGrade)을 따른다.
 */
export function selfGradeStep(detail: CheckDetail): SelfGradeStep | null {
  if (detail.session.grader !== "self" || detail.pending || !detail.current) { return null; }
  if (detail.current.canSelfGrade) { return "grade"; }
  if (detail.current.canAnswer) { return "answer"; }
  return null;
}

/** 화면에 카드로 보이는 문항: 끝났거나, 지금 문항이거나, 문답이 있는 문항. */
export function isShownItem(item: CheckItem, currentId: string | null, turnCount: number): boolean {
  return item.done || item.id === currentId || turnCount > 0;
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
  // 자기 채점은 스스로 고른 판정이라 재채점 대상이 아니다.
  if (judged.grader === "self") { return null; }
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

/** 한 줄 시스템 표기로 보일 턴 — '그냥 알려줘'(reveal)로 답을 공개한 튜터 턴(본문 없이 모범답 섹션에 답이 있다). */
export function isRevealNotice(turn: CheckTurn): boolean {
  return turn.role === "tutor" && turn.move === "reveal" && turn.status === "ready";
}

/** 튜터 턴 본문 — '그냥 알려줘'(reveal)는 모범답 섹션에만 답을 두고 턴에는 판정 배지만 남긴다. */
export function turnBody(turn: CheckTurn): string {
  return turn.move === "reveal" ? "" : turn.text;
}

/** 모범답·기준 공개 — 끝난 문항, 또는 자기 채점에서 답을 보낸 현재 문항(서버가 answer 를 준다). */
export function showItemReview(item: CheckItem): boolean {
  return item.done || Boolean(item.answer);
}

/** 하단 입력창 — active 세션에 현재 문항이 있을 때만 둔다. */
export function showComposer(detail: CheckDetail): boolean {
  return detail.session.status === "active" && detail.current !== null;
}

/** '그냥 알려줘' — 서버가 허용(current.canReveal)하고 실행 중이 아닐 때만. */
export function showReveal(detail: CheckDetail): boolean {
  return !detail.pending && Boolean(detail.current?.canReveal);
}

/** 확신도를 묻는지 — 확인·복습 문항의 첫 답 전에만(v3 §3). */
export function asksConfidence(detail: CheckDetail): boolean {
  const kind = detail.session.kind;
  return (kind === "check" || kind === "review") && detail.current?.answers === 0;
}

/** 결과의 '착각' 묶음 — 확실하다고 했는데 틀린 목표 줄. */
export function confidentErrors(rows: ResultRow[]): ResultRow[] {
  return rows.filter((r) => r.confidentError);
}

/** 보내기 — 답할 차례이고 실행 중이 아니며 1~최대 글자. */
export function canSend(detail: CheckDetail, text: string): boolean {
  const trimmed = text.trim();
  return !detail.pending && Boolean(detail.current?.canAnswer) && trimmed.length > 0 && trimmed.length <= CHECK_MAX_ANSWER_CHARS;
}

export interface ResultRow extends CheckResultItem {
  text: string;
  queued: boolean;
  /** 같은 목표의 처음 예측(예측 질문 답, 없으면 null) */
  prediction: Prediction | null;
  /** 이 세션에서 그 문항에 처음 쓴 답(지금 답) */
  answer: string;
  /** 그 문항 판정에서 드러난 오개념(턴별, 같은 글은 한 번) */
  misconceptions: RevealedMisconception[];
}

/** 판정 턴에서 드러난 오개념 — '맞아요, 그렇게 알았어요'는 그 턴 id 로 인정한다. */
export interface RevealedMisconception {
  turnId: string;
  text: string;
}

/** 그 문항의 판정 턴(튜터·완료·이의 제외)에 붙은 오개념, seq 순·같은 글은 처음 턴만. */
export function itemMisconceptions(turns: CheckTurn[], itemId: string): RevealedMisconception[] {
  const seen = new Set<string>();
  const out: RevealedMisconception[] = [];
  for (const t of [...turns].sort((a, b) => a.seq - b.seq)) {
    const text = t.grade?.misconception?.trim() ?? "";
    if (t.itemId !== itemId || t.role !== "tutor" || t.status !== "ready" || t.move === "dispute" || !text || seen.has(text)) { continue; }
    seen.add(text);
    out.push({ turnId: t.id, text });
  }
  return out;
}

export type MisconceptionButtonState = "idle" | "pending" | "acknowledged";

/** 오개념 버튼 상태 — 인정한 턴이면 '인정함', 보내는 중이면 pending. */
export function misconceptionButtonState(turnId: string, acknowledged: ReadonlySet<string>, pendingTurnId: string | null): MisconceptionButtonState {
  if (acknowledged.has(turnId)) { return "acknowledged"; }
  return pendingTurnId === turnId ? "pending" : "idle";
}

/** 기록만 하는 세션(예측 질문 · 배치 진단) — 판정·결과 요약이 없다. */
export function isRecorded(detail: CheckDetail): boolean {
  return detail.session.kind === "pretest" || detail.session.kind === "placement";
}

/** 그 문항에 학습자가 처음 쓴 답. */
export function firstAnswerOf(turns: CheckTurn[], itemId: string): string {
  const first = [...turns].sort((a, b) => a.seq - b.seq).find((t) => t.itemId === itemId && t.role === "learner" && t.move === "answer");
  return first?.text ?? "";
}

/** 결과 요약의 목표별 줄 — 통합 문항(objectiveId "")은 목표 레벨이 없으므로 뺀다. 처음 예측이 있으면 지금 답과 나란히. */
export function resultRows(detail: CheckDetail): ResultRow[] {
  if (isRecorded(detail)) { return []; }
  const mastery = new Map<string, ObjectiveMastery>((detail.mastery ?? []).map((m) => [m.objectiveId, m]));
  const predictions = new Map<string, Prediction>((detail.predictions ?? []).map((p) => [p.objectiveId, p]));
  return (detail.session.result?.items ?? [])
    .filter((r) => r.objectiveId !== "")
    .map((r) => {
      const m = mastery.get(r.objectiveId);
      return {
        ...r,
        text: m?.text || r.objectiveId,
        queued: r.queued ?? m?.queued ?? false,
        prediction: predictions.get(r.objectiveId) ?? null,
        answer: firstAnswerOf(detail.turns, r.itemId),
        misconceptions: itemMisconceptions(detail.turns, r.itemId),
      };
    });
}

/** 결과의 '처음 예측 → 지금 답' 줄 — 처음 예측과 지금 쓴 답이 모두 있을 때만. */
export function showPrediction(row: Pick<ResultRow, "prediction" | "answer">): boolean {
  return row.prediction !== null && row.answer.trim() !== "";
}

/** '다시 배우기' 제안 — 반복 실패(lapses ≥ 2, design §2.4) 또는 '이미 앎' 확인을 통과하지 못한 목표. */
export function suggestRelearn(row: CheckResultItem): boolean {
  return row.lapses >= 2 || Boolean(row.relearn);
}

/** '다시 볼 곳' 링크 글자 — 앵커가 가리키는 강의 섹션 제목(섹션·블록 앵커 모두). */
export function anchorTitle(sections: LectureSection[] | undefined, anchor: string): string {
  const section = (sections ?? []).find((s) => s.anchor === anchor || s.blocks.some((b) => b.anchor === anchor));
  return section?.title ?? "";
}
