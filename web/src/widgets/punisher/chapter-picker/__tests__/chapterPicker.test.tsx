import { fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

const replace = vi.fn();
const draftRes = {
  draftId: "d1",
  draft: {
    title: "웹 성능", summary: "",
    concepts: [
      { id: "k1", key: "k1", action: "create", name: "브라우저 렌더링", area: "web", depth: 1, summary: "", objectives: [{ text: "a", depth: 1 }], prerequisites: null },
      { id: "k2", key: "k2", action: "create", name: "CDN", area: "web", depth: 1, summary: "", objectives: [], prerequisites: null },
    ],
  },
};
const create = { mutate: vi.fn((_b: unknown, o?: { onSuccess?: (r: unknown) => void }) => o?.onSuccess?.(draftRes)), reset: vi.fn(), data: undefined as unknown, isPending: false, error: null };
const apply = { mutate: vi.fn((_id: string, o?: { onSuccess?: (r: unknown) => void }) => o?.onSuccess?.({ chapterId: "ch1" })), reset: vi.fn(), isPending: false, error: null };
const placement = { mutate: vi.fn((_b: unknown, o?: { onSuccess?: (r: unknown) => void }) => o?.onSuccess?.({ session: { id: "p9" } })), isPending: false, error: null };

vi.mock("@/src/shared/lib/router", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/src/shared/lib/router")>()),
  useRouter: () => ({ push: vi.fn(), replace, back: vi.fn() }),
}));
vi.mock("@/src/entities/punisher", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/src/entities/punisher")>()),
  useChapterSuggestions: () => ({ data: { question: "어떤 챕터를 열까요?", suggestions: [{ id: "s1", kind: "deepen", title: "웹 성능", reason: "HTTP 캐시 다음", score: 1 }] }, isLoading: false, error: null, refetch: vi.fn() }),
  useGraph: () => ({ data: { nodes: [] } }),
  useCreateChapterDraft: () => create,
  useApplyChapterDraft: () => apply,
  useStartDiagnostic: () => placement,
}));

const { ChapterPickerWidget } = await import("../ChapterPickerWidget");

describe("새 챕터 T-steps", () => {
  beforeEach(() => {
    window.localStorage.clear();
    replace.mockClear();
    create.data = undefined;
  });

  it("제안 고르기 → 설계 → 반영 → 건너뛰기 → 끝 화면 [지도에서 보기](replace)", () => {
    render(<ChapterPickerWidget />);
    expect(screen.getByText("1/4 · 제안 고르기")).toBeInTheDocument();
    expect(screen.queryByRole("link", { name: /학습/ })).toBeNull();
    expect(screen.getByRole("button", { name: "설계 보기" })).toBeDisabled();
    fireEvent.click(screen.getByRole("checkbox", { name: /웹 성능/ }));
    fireEvent.click(screen.getByRole("button", { name: "설계 보기" }));

    expect(screen.getByText("2/4 · 설계")).toBeInTheDocument();
    expect(screen.getByText("브라우저 렌더링")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "지도에 반영" }));

    expect(screen.getByText("3/4 · 반영")).toBeInTheDocument();
    expect(screen.getByText("개념 2개가 지도에 생겼어요")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "건너뛰기" }));

    expect(screen.getByText("‘웹 성능’ 챕터를 열었어요")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "지도에서 보기" }));
    expect(replace).toHaveBeenCalledWith("/map?chapter=ch1");
  });

  it("반영 뒤 [배치 확인]은 배치 확인 세션을 시작해 replace", () => {
    render(<ChapterPickerWidget />);
    fireEvent.click(screen.getByRole("checkbox", { name: /웹 성능/ }));
    fireEvent.click(screen.getByRole("button", { name: "설계 보기" }));
    fireEvent.click(screen.getByRole("button", { name: "지도에 반영" }));
    fireEvent.click(screen.getByRole("button", { name: "배치 확인" }));
    expect(placement.mutate).toHaveBeenCalledWith({ kind: "placement", chapterId: "ch1" }, expect.anything());
    expect(replace).toHaveBeenCalledWith("/check/p9");
  });

  it("설계 단계의 [이전]은 제안 고르기로 돌아간다", () => {
    render(<ChapterPickerWidget />);
    fireEvent.click(screen.getByRole("checkbox", { name: /웹 성능/ }));
    fireEvent.click(screen.getByRole("button", { name: "설계 보기" }));
    fireEvent.click(screen.getByRole("button", { name: "이전" }));
    expect(create.reset).toHaveBeenCalled();
  });
});
