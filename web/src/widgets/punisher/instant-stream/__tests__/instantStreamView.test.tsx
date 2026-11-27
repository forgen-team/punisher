import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { INITIAL_STAGE, type StageState } from "../lib/stageState";

let state: StageState = INITIAL_STAGE;
let titleSet: string | null | undefined;
const replace = vi.fn();

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
  return { ...actual, punisherApi: { ...actual.punisherApi, getJob: () => Promise.resolve({ startedAt: "" }) } };
});

const { InstantStreamView } = await import("../InstantStreamView");

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

  it("실패하면 실패 패널과 다시 만들기", () => {
    state = { ...INITIAL_STAGE, stage: "failed", reasons: ["검수 통과 못함"] };
    renderView();
    expect(screen.getByRole("alert")).toHaveTextContent("강의를 만들지 못했어요");
    expect(screen.getByRole("button", { name: "다시 만들기" })).toBeInTheDocument();
  });
});
