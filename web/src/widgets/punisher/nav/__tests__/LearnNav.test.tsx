import { render, screen } from "@testing-library/react";
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
    expect(screen.getByRole("tablist")).toBeTruthy();
    expect(screen.getByText("3")).toBeTruthy();
  });

  it("draws nothing off the section screens", () => {
    const { container } = renderAt("/settings");
    expect(container.innerHTML).toBe("");
  });
});
