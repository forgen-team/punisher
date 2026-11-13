// Typed fetch for the local Punisher server (same origin, /v1).
// Error envelope: {"error": {"code", "message", "requestId"}}.

export const PUNISHER_API_BASE = "/v1";

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
  const res = await fetch(`${PUNISHER_API_BASE}${path}`, {
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

/** Korean, user-facing description of a punisher API error. */
export function describePunisherError(error: unknown): string {
  if (!(error instanceof PunisherApiError)) {
    return error instanceof Error && error.message ? `요청에 실패했습니다: ${error.message}` : "요청에 실패했습니다.";
  }
  switch (error.status) {
    case 404:
      return "요청한 항목을 찾을 수 없습니다.";
    case 409:
      return "같은 강의를 이미 생성하고 있습니다. 잠시 후 다시 시도해 주세요.";
    case 429:
      if (error.code === "BUDGET_EXCEEDED") { return "오늘의 생성 예산을 모두 사용했습니다. 내일 다시 시도해 주세요."; }
      return error.code === "LLM_RATE_LIMITED" ? error.message : "요청이 너무 많습니다. 잠시 후 다시 시도해 주세요.";
    case 502:
      return error.message || "AI 응답 생성에 실패했습니다. 다시 시도해 주세요.";
    case 503:
      // GENERATION_UNAVAILABLE: Claude Code 미설치/미로그인 — 서버가 고치는 방법을 한국어로 보낸다.
      return error.message || "Claude Code를 사용할 수 없어요. 설정 화면의 안내를 확인해 주세요.";
    default:
      return error.status >= 500 ? "서버 오류가 발생했습니다. 잠시 후 다시 시도해 주세요." : error.message;
  }
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
