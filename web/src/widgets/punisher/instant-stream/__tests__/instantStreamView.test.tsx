import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { INITIAL_STAGE, type StageState } from "../lib/stageState";

let state: StageState = INITIAL_STAGE;
let titleSet: string | null | undefined;
const replace = vi.fn();
const cancelJob = vi.fn();

vi.mock("@/src/shared/lib/router", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/src/shared/lib/router")>()),
  useRouter: () => ({ push: vi.fn(), replace, back: vi.fn() }),
}));
vi.mock("@/src/shared/lib/screen-chrome", () => ({
  usePageActions: vi.fn(),
  useScreenTitle: (title: string | null | undefined) => { titleSet = title; },
}));
vi.mock("../useInstantStream", () => ({ useInstantStream: () => state }));
vi.mock("../useRetryInstant", () => ({ useRetryInstant: () => ({ available: true, isPending: false, run: vi.fn() }) }));
vi.mock("@/src/features/punisher/pretest", () => ({ usePretest: () => ({ data: undefined }), PretestPanel: () => null }));
vi.mock("@/src/shared/api/punisher", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/src/shared/api/punisher")>();
  return { ...actual, punisherApi: { ...actual.punisherApi, getJob: () => Promise.resolve({ startedAt: "" }),
    cancelJob: (jobId: string) => cancelJob(jobId) } };
});

const { InstantStreamView } = await import("../InstantStreamView");
const { PunisherApiError } = await import("@/src/shared/api/punisher");

const renderView = () =>
  render(<QueryClientProvider client={new QueryClient()}><InstantStreamView jobId="j1" /></QueryClientProvider>);

describe("생성 중 화면", () => {
  it("틀 제목 '강의 만드는 중', 자체 상단 바 없이 진행 단계만 보인다", () => {
    state = { ...INITIAL_STAGE, stage: "reviewing" };
    renderView();
    expect(titleSet).toBe("강의 만드는 중");
    expect(screen.getByText("강의를 만들고 있어요")).toBeInTheDocument();
    expect(screen.getByRole("listitem", { current: "step" })).toHaveTextContent("검수 중");
    expect(screen.queryByRole("heading", { level: 1 })).toBeNull();
  });

  it("발행되면 강의로 replace", () => {
    state = { ...INITIAL_STAGE, stage: "published", lectureId: "L1" };
    renderView();
    expect(replace).toHaveBeenCalledWith("/lectures/L1");
  });

  it("[취소]를 누르면 확인 없이 바로 취소하고 오늘로 replace + 토스트", async () => {
    state = { ...INITIAL_STAGE, stage: "writing" };
    replace.mockClear();
    cancelJob.mockReset().mockResolvedValue({ id: "j1", status: "failed", reason: "cancelled" });
    renderView();
    fireEvent.click(screen.getByRole("button", { name: "취소" }));
    await waitFor(() => expect(replace).toHaveBeenCalledWith("/"));
    expect(cancelJob).toHaveBeenCalledWith("j1");
    expect(screen.queryByRole("dialog")).toBeNull();
    expect(await screen.findByText("강의 만들기를 취소했어요")).toBeInTheDocument();
  });

  it("취소가 실패하면 그 자리에 인라인 오류, 이동하지 않는다", async () => {
    state = { ...INITIAL_STAGE, stage: "reviewing" };
    replace.mockClear();
    cancelJob.mockReset().mockRejectedValue(new Error("network"));
    renderView();
    fireEvent.click(screen.getByRole("button", { name: "취소" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("취소하지 못했어요. 다시 시도해 주세요");
    expect(replace).not.toHaveBeenCalled();
  });

  it("이미 끝난 작업(409)이면 취소할 수 없다고 알린다", async () => {
    state = { ...INITIAL_STAGE, stage: "rewriting", attempt: 1 };
    cancelJob.mockReset().mockRejectedValue(new PunisherApiError(409, "CONFLICT", "generation job is not running"));
    renderView();
    fireEvent.click(screen.getByRole("button", { name: "취소" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("이미 끝난 작업이라 취소할 수 없어요");
  });

  it("발행·실패 뒤에는 [취소]가 없다", () => {
    state = { ...INITIAL_STAGE, stage: "failed", reasons: ["x"] };
    renderView();
    expect(screen.queryByRole("button", { name: "취소" })).toBeNull();
  });

  it("실패하면 실패 패널과 다시 만들기", () => {
    state = { ...INITIAL_STAGE, stage: "failed", reasons: ["검수 통과 못함"] };
    renderView();
    expect(screen.getByRole("alert")).toHaveTextContent("강의를 만들지 못했어요");
    expect(screen.getByRole("button", { name: "다시 만들기" })).toBeInTheDocument();
  });
});
