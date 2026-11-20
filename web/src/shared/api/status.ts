import { useQuery } from "@tanstack/react-query";
import { punisherFetch } from "@/src/shared/api/punisher";

/** 강의 생성·검수·질문 답·채점을 모두 맡는 AI(설정 aiProvider). */
export type AiProvider = "claude" | "codex" | "agy";
export const AI_PROVIDERS: readonly AiProvider[] = ["claude", "codex", "agy"];
export const DEFAULT_AI_PROVIDER: AiProvider = "claude";

/** 화면에 보이는 이름(고유 명사라 언어마다 같다). */
export const AI_PROVIDER_NAMES: Record<AiProvider, string> = {
  claude: "Claude Code",
  codex: "Codex",
  agy: "Gemini (Antigravity)",
};

export function isAiProvider(value: unknown): value is AiProvider {
  return value === "claude" || value === "codex" || value === "agy";
}

export interface ClaudeStatus {
  ready: boolean;
  installed: boolean;
  version?: string;
  /** null when the installed CLI cannot report it. */
  loggedIn: boolean | null;
  code?: "claude_unavailable" | "claude_not_logged_in" | "claude_rate_limited";
  active: number;
  queued: number;
}

/** 제공자 하나의 상태 줄 — agy 는 로그인을 미리 알 수 없어 loggedIn 이 늘 null. */
export interface ProviderStatus {
  id: AiProvider;
  ready: boolean;
  installed: boolean;
  version?: string;
  loggedIn: boolean | null;
  /** `<id>_unavailable` · `<id>_not_logged_in` · `<id>_rate_limited` */
  code?: string;
}

export interface AppStatus {
  claude: ClaudeStatus;
  /** 고른 AI — 없으면 예전 서버(claude 만). */
  provider?: AiProvider;
  providers?: ProviderStatus[];
  /** 고른 AI 기준 안내(서버는 한국어). */
  message?: string;
  steps?: string[];
  appVersion: string;
  dataDir: string;
}

/** 고른 AI id — 예전 서버·모르는 값이면 claude. */
export function selectedProviderId(status: AppStatus | undefined): AiProvider {
  return isAiProvider(status?.provider) ? status.provider : DEFAULT_AI_PROVIDER;
}

/** 제공자 목록 — 예전 서버면 claude 상태 하나로 만든다. */
export function providerStatuses(status: AppStatus): ProviderStatus[] {
  if (status.providers && status.providers.length > 0) { return status.providers.filter((p) => isAiProvider(p.id)); }
  const { ready, installed, version, loggedIn, code } = status.claude;
  return [{ id: "claude", ready, installed, version, loggedIn, code }];
}

/** 고른 AI 의 상태(목록에 없으면 undefined). */
export function selectedProviderStatus(status: AppStatus): ProviderStatus | undefined {
  const id = selectedProviderId(status);
  return providerStatuses(status).find((p) => p.id === id);
}

/** useAppStatus 의 쿼리 키 — 제공자를 바꾼 뒤 캐시를 바로 고칠 때 쓴다. */
export const APP_STATUS_KEY = ["punisher", "status"] as const;

/** GET /v1/status — local AI CLI readiness for every provider (polled; never calls the model). */
export function useAppStatus() {
  return useQuery({
    queryKey: APP_STATUS_KEY,
    queryFn: () => punisherFetch<AppStatus>("/status"),
    staleTime: 30_000,
    refetchInterval: 60_000,
    refetchOnWindowFocus: true,
  });
}

