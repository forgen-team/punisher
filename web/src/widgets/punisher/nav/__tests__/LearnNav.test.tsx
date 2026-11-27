import { render, screen, within } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { MemoryRouter } from "react-router-dom";
import { LearnNav } from "../LearnNav";

const state = vi.hoisted(() => ({ restToday: false }));

vi.mock("@/src/entities/punisher", () => ({
  useActivity: () => ({ data: { restToday: state.restToday, today: { reviewsDue: 2, cardsDue: 1 } } }),
  useToday: () => ({ data: { items: [], exhausted: false, unchecked: [{ lectureId: "l1" }, { lectureId: "l2" }] } }),
}));

function renderAt(path: string) {
  return render(
    <MemoryRouter initialEntries={[path]}>
      <LearnNav />
    </MemoryRouter>,
  );
}

describe("LearnNav (standalone section tabs)", () => {
  it("draws 오늘·지도·기록 with 복습 + 확인 대기 + 카드 next to 오늘", () => {
    state.restToday = false;
    renderAt("/map");
    const nav = screen.getByRole("navigation", { name: "punisher 섹션" });
    const links = within(nav).getAllByRole("link");
    expect(links.map((a) => a.textContent?.replace(/\d+/g, ""))).toEqual(["오늘", "지도", "기록"]);
    expect(within(nav).getByRole("link", { name: "지도" })).toHaveAttribute("aria-current", "page");
    // 2(복습) + 1(카드) + 2(확인 대기)
    expect(within(within(nav).getByRole("link", { name: /오늘/ })).getByText("5")).toBeInTheDocument();
    // 탭 줄에는 제목을 두지 않는다(제목은 Page h1)
    expect(screen.queryByRole("heading")).toBeNull();
  });

  it("marks 오늘 on screens that belong to it (lecture, review session …)", () => {
    renderAt("/lectures/abc");
    const nav = screen.getByRole("navigation", { name: "punisher 섹션" });
    expect(within(nav).getByRole("link", { name: /오늘/ })).toHaveAttribute("aria-current", "page");
  });

  it("stays visible off the sections, with nothing marked current", () => {
    renderAt("/settings");
    const nav = screen.getByRole("navigation", { name: "punisher 섹션" });
    expect(within(nav).getAllByRole("link")).toHaveLength(3);
    for (const link of within(nav).getAllByRole("link")) { expect(link).not.toHaveAttribute("aria-current"); }
  });

  it("shows no number on a rest day", () => {
    state.restToday = true;
    renderAt("/");
    const nav = screen.getByRole("navigation", { name: "punisher 섹션" });
    expect(within(nav).getByRole("link", { name: "오늘" })).toBeInTheDocument();
    state.restToday = false;
  });
});
