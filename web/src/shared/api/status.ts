import { useQuery } from "@tanstack/react-query";
import { punisherFetch } from "@/src/shared/api/punisher";

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

export interface AppStatus {
  claude: ClaudeStatus;
  message?: string;
  steps?: string[];
  appVersion: string;
  dataDir: string;
}

/** GET /v1/status — local Claude Code CLI readiness (polled; never calls the model). */
export function useAppStatus() {
  return useQuery({
    queryKey: ["punisher", "status"],
    queryFn: () => punisherFetch<AppStatus>("/status"),
    staleTime: 30_000,
    refetchInterval: 60_000,
    refetchOnWindowFocus: true,
  });
}
