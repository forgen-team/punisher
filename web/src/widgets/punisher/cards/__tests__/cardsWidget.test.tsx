import { act, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
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
const remove = { mutate: vi.fn((_id: string, o?: { onSuccess?: () => void }) => o?.onSuccess?.()), error: null };
const restore = { mutateAsync: vi.fn((_id: string) => Promise.resolve()) };

vi.mock("@/src/entities/punisher", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/src/entities/punisher")>()),
  useCards: (dueOnly: boolean) => ({ data: dueOnly ? due : all, isLoading: false, error: null, refetch: vi.fn() }),
  useGradeCard: () => grade,
  useDeleteCard: () => remove,
  useRestoreCard: () => restore,
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

describe("모든 카드 — 삭제 되돌리기", () => {
  async function pickDelete(question: string) {
    renderAt("/cards?tab=all");
    const trigger = await screen.findByRole("button", { name: `카드 메뉴: ${question}` });
    act(() => {
      fireEvent.pointerDown(trigger, { button: 0, pointerType: "mouse" });
      fireEvent.click(trigger);
    });
    const menu = await screen.findByRole("menu", { name: `카드 메뉴: ${question}` });
    fireEvent.click(within(menu).getByRole("menuitem", { name: "카드 삭제" }));
  }

  it("⋯ 카드 삭제 → 확인 창 없이 바로 삭제 + '카드를 삭제했어요 [되돌리기]'", async () => {
    await pickDelete("셋째 카드");
    expect(remove.mutate).toHaveBeenCalledWith("c3", expect.anything());
    expect(screen.queryByRole("alertdialog")).toBeNull();
    expect(screen.queryByRole("dialog")).toBeNull();
    expect(await screen.findByText("카드를 삭제했어요")).toBeInTheDocument();
    expect(restore.mutateAsync).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole("button", { name: "되돌리기" }));
    await waitFor(() => expect(restore.mutateAsync).toHaveBeenCalledWith("c3"));
  });

  it("되돌리기가 실패하면 '되돌리지 못했어요'", async () => {
    restore.mutateAsync.mockRejectedValueOnce(new Error("boom"));
    await pickDelete("첫 카드");
    fireEvent.click(await screen.findByRole("button", { name: "되돌리기" }));
    expect(await screen.findByText("되돌리지 못했어요")).toBeInTheDocument();
  });
});
