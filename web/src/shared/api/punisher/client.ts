// Typed fetch for the local Punisher server (same origin, /v1).
// Error envelope: {"error": {"code", "message", "requestId"}}.

import { apiBase } from "@/src/shared/config/runtime";
import { getLocale, translate } from "@/src/shared/i18n";
import { messages } from "./i18n";
import { errorCodeMessage } from "./reasons";

/** API base ("/v1" standalone, "/apps-api/punisher/v1" inside Witems). */
export function punisherApiBase(): string {
  return apiBase();
}

export class PunisherApiError extends Error {
  readonly status: number;
  readonly code: string;
  readonly requestId?: string;
  /** 검증 실패 상세(예: CHAPTER_INVALID 의 problems) */
  readonly problems: string[];

  constructor(status: number, code: string, message: string, requestId?: string, problems: string[] = []) {
    super(message);
    this.name = "PunisherApiError";
    this.status = status;
    this.code = code;
    this.requestId = requestId;
    this.problems = problems;
  }
}

interface ErrorEnvelope {
  error?: { code?: string; message?: string; requestId?: string; problems?: string[] };
}

async function parseError(res: Response): Promise<PunisherApiError> {
  let envelope: ErrorEnvelope = {};
  try {
    envelope = (await res.json()) as ErrorEnvelope;
  } catch {
    envelope = {};
  }
  const err = envelope.error ?? {};
  return new PunisherApiError(res.status, err.code ?? "HTTP_ERROR", err.message ?? `HTTP ${res.status}`, err.requestId, err.problems ?? []);
}

export interface PunisherRequestInit extends Omit<RequestInit, "body"> {
  json?: unknown;
}

/** Calls `/v1{path}` and returns parsed JSON (undefined for 204). */
export async function punisherFetch<T>(path: string, init: PunisherRequestInit = {}): Promise<T> {
  const { json, headers, ...rest } = init;
  const res = await fetch(`${punisherApiBase()}${path}`, {
      ...rest,
    headers: {
      Accept: "application/json",
      ...(json !== undefined ? { "Content-Type": "application/json" } : {}),
      ...headers,
    },
    body: json !== undefined ? JSON.stringify(json) : undefined,
  });
  if (!res.ok) {
    throw await parseError(res);
  }
  if (res.status === 204) { return undefined as T; }
  return (await res.json()) as T;
}

/** User-facing description of a punisher API error, in the current locale. */
export function describePunisherError(error: unknown): string {
  const t = translate(messages);
  if (!(error instanceof PunisherApiError)) {
    return error instanceof Error && error.message ? t.requestFailedWith(error.message) : t.requestFailed;
  }
  const byCode = errorCodeMessage(error.code);
  if (byCode) { return byCode; }
  switch (error.status) {
    case 404:
      return t.notFound;
    case 409:
      return t.alreadyGenerating;
    case 429:
      if (error.code === "BUDGET_EXCEEDED") { return t.budgetExceeded; }
      // LLM_RATE_LIMITED: 로컬 Claude 사용 한도 — 서버가 한국어 안내를 보낸다(ko 일 때만 그대로).
      return error.code === "LLM_RATE_LIMITED" ? localServerText(error.message, t.claudeRateLimited) : t.tooManyRequests;
    case 502:
      return localServerText(error.message, t.aiFailed(""));
    case 503:
      // GENERATION_UNAVAILABLE: Claude Code 미설치/미로그인 — 서버가 고치는 방법을 한국어로 보낸다.
      return localServerText(error.message, t.claudeUnavailable);
    default:
      return error.status >= 500 ? t.serverError : error.message;
  }
}

/**
 * 단독판: 로컬 Claude CLI 안내(services.DescribeLLMError)는 서버가 한국어로만 쓴다 —
 * ko 화면이면 그 문구, 아니면 같은 뜻의 화면 문구.
 */
function localServerText(serverMessage: string, fallback: string): string {
  return getLocale() === "ko" && serverMessage ? serverMessage : fallback;
}

function shouldRetry(failureCount: number, error: unknown): boolean {
  if (error instanceof PunisherApiError && error.status < 500) { return false; }
  return failureCount < 1;
}

/** Project defaults for punisher queries (CLAUDE.md: staleTime 1m, retry 1, no focus refetch). */
export const punisherQueryDefaults = {
  staleTime: 60_000,
  retry: shouldRetry,
  refetchOnWindowFocus: false,
} as const;
