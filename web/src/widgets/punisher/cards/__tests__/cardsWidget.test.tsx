import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter, useLocation } from "react-router-dom";
import { describe, expect, it, vi } from "vitest";
import type { ReviewCard } from "@/src/shared/api/punisher";

const card = (id: string, question: string): ReviewCard => ({
  id, sourceKind: "note", sourceId: "n1", lectureId: "", conceptId: "", anchor: "", question, answer: `${question} 답`,
  reps: 0, lapses: 0, nextReviewAt: "2026-11-29T00:00:00+09:00", suspendedAt: null, createdAt: "", sourceUrl: "", externalId: "", stale: false,
});
const due = [card("c1", "첫 카드"), card("c2", "둘째 카드")];
const all = [...due, card("c3", "셋째 카드")];
const grade = { mutate: vi.fn((_v: unknown, o?: { onSuccess?: () => void }) => o?.onSuccess?.()), isPending: false, error: null, variables: undefined };
const remove = { mutate: vi.fn(), error: null };

vi.mock("@/src/entities/punisher", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/src/entities/punisher")>()),
  useCards: (dueOnly: boolean) => ({ data: dueOnly ? due : all, isLoading: false, error: null, refetch: vi.fn() }),
  useGradeCard: () => grade,
  useDeleteCard: () => remove,
}));

const { CardsWidget } = await import("../CardsWidget");

function Where() {
  const loc = useLocation();
  return <p data-testid="where">{loc.pathname + loc.search}</p>;
}

function renderAt(url: string) {
  return render(<MemoryRouter initialEntries={[url]}><CardsWidget /><Where /></MemoryRouter>);
}

describe("카드 복습", () => {
  it("복습 탭은 앞면 → [답 보기] → [다시][알았음] → 끝에 [오늘로]", () => {
    renderAt("/cards");
    expect(screen.getByRole("link", { name: /복습/ })).toHaveAttribute("aria-current", "page");
    expect(screen.getByText("1/2")).toBeInTheDocument();
    expect(screen.getByText("첫 카드")).toBeInTheDocument();
    expect(screen.queryByText("첫 카드 답")).toBeNull();
    expect(screen.queryByRole("button", { name: "알았음" })).toBeNull();
    fireEvent.click(screen.getByRole("button", { name: "답 보기" }));
    expect(screen.getByText("첫 카드 답")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "답 보기" })).toBeNull();
    expect(screen.getByRole("button", { name: "다시" })).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "알았음" }));
    expect(grade.mutate).toHaveBeenCalledWith({ id: "c1", rating: 3 }, expect.anything());
    expect(screen.getByText("2/2")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "답 보기" }));
    fireEvent.click(screen.getByRole("button", { name: "다시" }));
    expect(screen.getByText("카드 복습을 마쳤어요")).toBeInTheDocument();
    expect(screen.getByText("알았음 1 · 다시 1")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "오늘로" })).toHaveAttribute("href", "/");
  });

  it("탭을 누르면 ?tab=all 로 replace 하고 모든 카드 목록을 보인다", async () => {
    renderAt("/cards");
    const tab = screen.getByRole("link", { name: "모든 카드" });
    expect(tab).toHaveAttribute("href", "/cards?tab=all");
    fireEvent.click(tab);
    await waitFor(() => expect(screen.getByTestId("where").textContent).toBe("/cards?tab=all"));
    expect(screen.getByText("셋째 카드")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "답 보기" })).toBeNull();
  });
});
