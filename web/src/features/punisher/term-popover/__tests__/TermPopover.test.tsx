import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

const push = vi.fn();
const createStudyRequest = vi.fn((_query: string, options?: { onSuccess?: () => void }) => options?.onSuccess?.());

vi.mock("@/src/shared/lib/router", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/src/shared/lib/router")>()),
  useRouter: () => ({ push, replace: vi.fn(), back: vi.fn() }),
}));
vi.mock("@/src/entities/punisher", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/src/entities/punisher")>()),
  useGloss: () => ({ data: { gloss: "응답 내용에 붙는 판 번호예요." }, isLoading: false, error: null }),
  useCreateStudyRequest: () => ({ mutate: createStudyRequest, isPending: false, isSuccess: false, error: null }),
}));
vi.mock("../../instant-start", () => ({ useLaunchInstant: () => ({ launch: vi.fn(), isPending: false, error: null }) }));
vi.mock("../../reading-assist/ReadingThreadsProvider", () => ({
  useReadingThreads: () => ({ start: vi.fn(), getAnchor: () => "s1" }),
}));

const { TermPopover } = await import("../TermPopover");

describe("용어 팝오버", () => {
  it("[더 깊게][더 쉽게][이어서 묻기][나중에 공부] 순서", () => {
    render(<TermPopover term="ETag" conceptId="c9" onBeforeDeeper={() => undefined} />);
    fireEvent.click(screen.getByRole("button", { name: "ETag" }));
    const labels = screen.getAllByRole("button").map((b) => b.textContent);
    const order = ["더 깊게", "더 쉽게", "이어서 묻기", "나중에 공부"].map((l) => labels.indexOf(l));
    expect(order.every((i) => i >= 0)).toBe(true);
    expect([...order].sort((a, b) => a - b)).toEqual(order);
  });

  it("[나중에 공부] → 공부할 주제에 넣고 토스트 [오늘에서 보기] 로 오늘 화면", async () => {
    render(<TermPopover term="ETag" conceptId="c9" onBeforeDeeper={() => undefined} />);
    fireEvent.click(screen.getByRole("button", { name: "ETag" }));
    fireEvent.click(screen.getByRole("button", { name: "나중에 공부" }));
    expect(createStudyRequest).toHaveBeenCalledWith("ETag", expect.anything());
    expect(await screen.findByText("공부할 주제에 넣었어요")).toBeInTheDocument();
    act(() => fireEvent.click(screen.getByRole("button", { name: "오늘에서 보기" })));
    await waitFor(() => expect(push).toHaveBeenCalledWith("/"));
  });
});
