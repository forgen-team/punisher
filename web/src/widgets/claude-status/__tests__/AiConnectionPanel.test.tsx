import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { APP_STATUS_KEY, type AppStatus } from "@/src/shared/api/status";

const refetch = vi.fn();
const mutate = vi.fn((_patch: unknown, opts?: { onSuccess?: () => void; onSettled?: () => void }) => {
  opts?.onSuccess?.();
  opts?.onSettled?.();
});

const status: AppStatus = {
  claude: { ready: true, installed: true, version: "2.1.0", loggedIn: true, active: 0, queued: 0 },
  provider: "claude",
  providers: [
    { id: "claude", ready: true, installed: true, version: "2.1.0", loggedIn: true },
    { id: "codex", ready: true, installed: true, version: "0.50.0", loggedIn: true },
    { id: "agy", ready: false, installed: false, loggedIn: null, code: "agy_unavailable" },
  ],
  appVersion: "1.0.0",
  dataDir: "/tmp/punisher",
};

vi.mock("@/src/shared/api/status", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/src/shared/api/status")>()),
  useAppStatus: () => ({ data: status, isLoading: false, isFetching: false, refetch }),
}));

vi.mock("@/src/entities/punisher", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/src/entities/punisher")>()),
  useSaveLearningSettings: () => ({ mutate, isPending: false, variables: undefined, error: null }),
}));

const { AiConnectionPanel } = await import("../AiConnectionPanel");

describe("설정 › AI 연결", () => {
  it("세 제공자를 보여 주고, 설치 안 된 것은 고를 수 없고, 바꾸면 aiProvider 를 저장한 뒤 상태를 다시 받는다", () => {
    const client = new QueryClient();
    client.setQueryData(APP_STATUS_KEY, status);
    render(
      <QueryClientProvider client={client}>
        <AiConnectionPanel />
      </QueryClientProvider>,
    );
    const claude = screen.getByRole("radio", { name: /Claude Code/ });
    const codex = screen.getByRole("radio", { name: /Codex/ });
    const agy = screen.getByRole("radio", { name: /Gemini \(Antigravity\)/ });
    expect(screen.getAllByRole("radio")).toHaveLength(3);
    expect(claude).toBeChecked();
    expect(screen.getByText("사용 중")).toBeInTheDocument();
    expect(agy).toBeDisabled();
    expect(screen.getByText("설치 안 됨")).toBeInTheDocument();
    expect(screen.getByText("설치됨 · 0.50.0 · 로그인됨")).toBeInTheDocument();
    expect(codex).toBeEnabled();

    fireEvent.click(codex);
    expect(mutate).toHaveBeenCalledWith({ aiProvider: "codex" }, expect.anything());
    expect(refetch).toHaveBeenCalled();
    // 저장 성공 직후 다시 받기 전에도 고른 AI 가 예전 것으로 돌아가 보이지 않게 캐시를 먼저 바꾼다.
    expect(client.getQueryData<AppStatus>(APP_STATUS_KEY)?.provider).toBe("codex");
    expect(screen.getByRole("button", { name: "다시 확인" })).toBeInTheDocument();
  });
});
