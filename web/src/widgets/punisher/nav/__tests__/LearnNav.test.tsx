import { render, screen, within } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { MemoryRouter } from "react-router-dom";
import { LearnNav } from "../LearnNav";

vi.mock("@/src/entities/punisher", () => ({
  useActivity: () => ({ data: { restToday: false, today: { reviewsDue: 2, cardsDue: 1 } } }),
}));

function renderAt(path: string) {
  return render(
    <MemoryRouter initialEntries={[path]}>
      <LearnNav />
    </MemoryRouter>,
  );
}

describe("LearnNav (standalone)", () => {
  it("draws the section tabs with the due count on 강의", () => {
    renderAt("/map");
    // 탭은 링크 내비(nav + a[aria-current]) — 섹션 3개, 지금 화면(지도)이 현재 탭
    const nav = screen.getByRole("navigation", { name: "학습 보기" });
    expect(within(nav).getAllByRole("link")).toHaveLength(3);
    expect(within(nav).getByRole("link", { name: "지도" })).toHaveAttribute("aria-current", "page");
    // 복습 수(2 + 1)는 '강의' 탭에 붙는다
    expect(within(within(nav).getByRole("link", { name: /강의/ })).getByText("3")).toBeInTheDocument();
  });

  it("draws nothing off the section screens", () => {
    renderAt("/settings");
    // 전역 render 래퍼의 FeedbackProvider 알림 영역은 남으므로, LearnNav 가 그리는 것(제목·탭 내비·링크)이 없음을 본다
    expect(screen.queryByRole("navigation")).toBeNull();
    expect(screen.queryByRole("heading")).toBeNull();
    expect(screen.queryAllByRole("link")).toHaveLength(0);
  });
});
