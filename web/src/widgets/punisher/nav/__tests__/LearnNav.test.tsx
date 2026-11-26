import { render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { MemoryRouter } from "react-router-dom";
import { setWitemsRuntime, type WitemsRuntime } from "@/src/shared/config/runtime";
import { LearnNav } from "../LearnNav";

vi.mock("@/src/entities/punisher", () => ({
  useActivity: () => ({ data: { restToday: false, today: { reviewsDue: 2, cardsDue: 1 } } }),
}));

function runtime(extra: Partial<WitemsRuntime> = {}): WitemsRuntime {
  return { apiBase: "/apps-api/punisher/", portal: document.body, setTitle: () => undefined, setActions: () => undefined, ...extra };
}

function renderAt(path: string) {
  return render(
    <MemoryRouter initialEntries={[path]}>
      <LearnNav />
    </MemoryRouter>,
  );
}

describe("LearnNav", () => {
  afterEach(() => setWitemsRuntime(null));

  it("keeps the in-app tabs standalone", () => {
    renderAt("/map");
    expect(screen.getByRole("tablist")).toBeTruthy();
    expect(screen.getByText("3")).toBeTruthy();
  });

  it("leaves the tabs to the shell sidebar sub-menu and sends the due count as the 강의 badge", () => {
    const setBadges = vi.fn();
    setWitemsRuntime(runtime({ setBadges }));
    const { container } = renderAt("/map");
    expect(container.innerHTML).toBe("");
    expect(setBadges).toHaveBeenLastCalledWith({ "#/": 3 });
  });

  it("keeps the tabs in a shell without the sub-menu", () => {
    setWitemsRuntime(runtime());
    renderAt("/");
    expect(screen.getByRole("tablist")).toBeTruthy();
  });
});
