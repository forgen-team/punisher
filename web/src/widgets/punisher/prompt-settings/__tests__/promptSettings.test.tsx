import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { PromptBlock } from "@/src/shared/api/punisher";

const block = (id: string, title: string, over: Partial<PromptBlock> = {}): PromptBlock => ({
  id, stage: "outline", stageTitle: "개요 짜기", title, purpose: `${title} 목적`, appliesTo: "강의", advanced: false, vars: [], default: "기본 본문", body: "내 본문", overridden: true, ...over,
});
const blocks = [block("b1", "개요"), block("b2", "본문 쓰기", { stage: "lecture", stageTitle: "본문 쓰기" })];
const change = { mutate: vi.fn(), isPending: false, error: null, variables: undefined };
let split = false;
let titleSet: string | null | undefined;

vi.mock("@forgen-team/ui", async (importOriginal) => ({ ...(await importOriginal<typeof import("@forgen-team/ui")>()), useIsSplit: () => split }));
vi.mock("@/src/shared/lib/screen-chrome", () => ({
  usePageActions: vi.fn(),
  useScreenTitle: (title: string | null | undefined) => { titleSet = title; },
}));
vi.mock("@/src/entities/punisher", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/src/entities/punisher")>()),
  usePrompts: () => ({ data: { blocks, feedbackHints: {}, version: "1" }, isLoading: false, error: null, refetch: vi.fn() }),
  useChangePrompt: () => change,
  usePromptHistory: () => ({ data: [], isLoading: false, error: null }),
}));

const { PromptSettingsWidget } = await import("../PromptSettingsWidget");

const renderAt = (url: string) => render(<MemoryRouter initialEntries={[url]}><PromptSettingsWidget /></MemoryRouter>);

describe("강의 프롬프트", () => {
  beforeEach(() => { window.localStorage.clear(); change.mutate.mockClear(); titleSet = undefined; });

  it("폰: 목록은 화면 제목을 바꾸지 않고, ?block= 전체 화면은 제목이 블록 이름", () => {
    split = false;
    const { unmount } = renderAt("/settings/prompts");
    expect(titleSet).toBeUndefined();
    expect(screen.getByText("개요 목적")).toBeInTheDocument();
    unmount();
    renderAt("/settings/prompts?block=b2");
    expect(titleSet).toBe("본문 쓰기");
    expect(screen.queryByText("개요 목적")).toBeNull();
    expect(screen.queryByRole("link", { name: /설정|강의 프롬프트/ })).toBeNull();
  });

  it("데스크톱: 목록 + 편집 2열, 제목은 틀 몫", () => {
    split = true;
    renderAt("/settings/prompts?block=b1");
    expect(titleSet).toBeUndefined();
    expect(screen.getByRole("navigation", { name: "프롬프트 블록" })).toBeInTheDocument();
    expect(screen.getByRole("textbox", { name: "개요 본문" })).toBeInTheDocument();
  });

  it("'기본값으로 되돌리기'는 확인 창을 거친다", async () => {
    split = true;
    renderAt("/settings/prompts?block=b1");
    fireEvent.click(screen.getByRole("button", { name: "기본값으로 되돌리기" }));
    const dialog = await screen.findByRole("alertdialog");
    expect(dialog).toHaveTextContent("‘개요’ 블록을 기본값으로 되돌릴까요?");
    expect(change.mutate).not.toHaveBeenCalled();
    fireEvent.click(within(dialog).getByRole("button", { name: "기본값으로 되돌리기" }));
    await waitFor(() => expect(change.mutate).toHaveBeenCalledWith({ kind: "reset", id: "b1" }, expect.anything()));
  });
});
