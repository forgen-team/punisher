/**
 * 즉석 강의 생성 단계(v2.2 §2.1) — SSE 이벤트 → 화면 상태.
 * 새 계약: `stage` 이벤트 {"type":"stage","stage":"writing|checking|reviewing|rewriting|published|failed","attempt","maxAttempts","lectureId"?,"reasons"?,"reasonCodes"?}.
 * reasonCodes 가 모두 아는 code 면 화면 언어로 옮긴 문구를, 아니면 서버 문구(reasons)를 쓴다.
 * 예전 서버 이벤트(status·delta·done·error)도 같은 상태로 바꾼다(본문 delta 는 보이지 않으므로 버린다).
 */
import { translate, type Locale } from "@/src/shared/i18n";
import { translateReasonCodes } from "@/src/shared/api/punisher";
import { messages } from "../i18n";

export type GenStage = "connecting" | "writing" | "checking" | "reviewing" | "rewriting" | "published" | "failed";

export interface StageState {
  stage: GenStage;
  /** 고쳐 쓰기 회차(1부터, rewriting 일 때 의미) */
  attempt: number;
  maxAttempts: number;
  lectureId?: string;
  /** 실패·재작성 사유(검수 지적 요약) */
  reasons: string[];
}

export const DEFAULT_MAX_ATTEMPTS = 2;

export const INITIAL_STAGE: StageState = { stage: "connecting", attempt: 0, maxAttempts: DEFAULT_MAX_ATTEMPTS, reasons: [] };

export type StreamEventName = "stage" | "status" | "delta" | "done" | "error" | "message";

const STAGES = new Set<GenStage>(["writing", "checking", "reviewing", "rewriting", "published", "failed"]);

type Payload = Record<string, unknown>;

function strings(value: unknown): string[] {
  if (typeof value === "string") { return value ? [value] : []; }
  return Array.isArray(value) ? value.filter((v): v is string => typeof v === "string" && v.length > 0) : [];
}

function reasonsOf(data: Payload): string[] {
  const translated = translateReasonCodes(data.reasonCodes);
  return translated.length > 0 ? translated : strings(data.reasons);
}

function positive(value: unknown, fallback: number): number {
  return typeof value === "number" && Number.isFinite(value) && value > 0 ? Math.floor(value) : fallback;
}

function failed(prev: StageState, reasons: string[]): StageState {
  return { ...prev, stage: "failed", reasons: reasons.length > 0 ? reasons : prev.reasons };
}

function fromStage(prev: StageState, data: Payload): StageState {
  const stage = data.stage as GenStage;
  if (!STAGES.has(stage)) { return prev; }
  const maxAttempts = positive(data.maxAttempts, prev.maxAttempts);
  const attempt = stage === "rewriting" ? positive(data.attempt, prev.attempt + 1) : positive(data.attempt, prev.attempt);
  const reasons = reasonsOf(data);
  if (stage === "published") {
    const lectureId = typeof data.lectureId === "string" && data.lectureId ? data.lectureId : undefined;
    return lectureId ? { ...prev, stage, attempt, maxAttempts, lectureId } : failed(prev, []);
  }
  if (stage === "failed") { return { ...failed(prev, reasons), attempt, maxAttempts }; }
  return { ...prev, stage, attempt, maxAttempts, reasons: reasons.length > 0 ? reasons : prev.reasons };
}

/** 예전 status 이벤트: generating → writing, validating → checking, rewriting → 고쳐 쓰기 회차 +1 */
function fromLegacyStatus(prev: StageState, data: Payload): StageState {
  if (data.stage === "rewriting") {
    return { ...prev, stage: "rewriting", attempt: prev.attempt + 1, reasons: strings(data.reason) };
  }
  const stage: GenStage = data.stage === "validating" ? "checking" : "writing";
  return prev.stage === stage ? prev : { ...prev, stage };
}

/** 한 이벤트를 반영한 다음 상태. 끝난 상태(published·failed)는 더 바꾸지 않는다. 같은 상태면 prev 를 그대로 돌려준다. */
export function reduceStage(prev: StageState, event: StreamEventName, data: Payload | null): StageState {
  if (prev.stage === "published" || prev.stage === "failed") { return prev; }
  if (!data) { return prev; }
  if (event === "stage" || (event === "message" && data.type === "stage")) { return fromStage(prev, data); }
  switch (event) {
    case "status":
      return fromLegacyStatus(prev, data);
    case "delta":
      return prev.stage === "connecting" ? { ...prev, stage: "writing" } : prev;
    case "done":
      return typeof data.lectureId === "string" && data.lectureId
        ? { ...prev, stage: "published", lectureId: data.lectureId }
        : failed(prev, [translate(messages).notSaved]);
    case "error":
      return failed(prev, strings(data.reasons).length > 0 ? strings(data.reasons) : strings(data.message || data.code || translate(messages).generateFailed));
    default:
      return prev;
  }
}

/** 단계 이름(화면 표시). 고치는 중은 회차를 붙인다. */
export function stageLabel(state: StageState, locale: Locale): string {
  const t = translate(messages, locale);
  switch (state.stage) {
    case "connecting":
      return t.connecting;
    case "writing":
      return t.writing;
    case "checking":
    case "reviewing":
      return t.reviewing;
    case "rewriting":
      return t.rewriting(Math.max(1, state.attempt), state.maxAttempts);
    case "published":
      return t.published;
    case "failed":
      return t.failed;
  }
}

/** 단계 표시의 세 칸(작성 → 검수 → 고침) 중 지금 칸 */
export type StepKey = "write" | "review" | "fix";

export function currentStep(stage: GenStage): StepKey | null {
  if (stage === "writing") { return "write"; }
  if (stage === "checking" || stage === "reviewing") { return "review"; }
  if (stage === "rewriting") { return "fix"; }
  return null;
}

/** 실패 사유 요약 — 최대 3줄 */
export function failureLines(state: StageState): string[] {
  return state.reasons.slice(0, 3);
}
