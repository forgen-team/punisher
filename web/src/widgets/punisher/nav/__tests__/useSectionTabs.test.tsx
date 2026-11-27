import { fireEvent, render, screen, within } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { MemoryRouter, useLocation } from "react-router-dom";
import { AppFrame, UIProvider } from "@forgen-team/ui";
import { Link } from "@/src/shared/lib/router";
import { resetTabMemory } from "../lib/tabMemory";
import { useSectionTabs } from "../useSectionTabs";

const state = vi.hoisted(() => ({ restToday: false }));

vi.mock("@/src/entities/punisher", () => ({
  useActivity: () => ({ data: { restToday: state.restToday, today: { reviewsDue: 2, cardsDue: 1 } } }),
  useToday: () => ({ data: { items: [], exhausted: false, unchecked: [{ lectureId: "l1" }, { lectureId: "l2" }] } }),
}));

function Where() {
  const { pathname, search } = useLocation();
  return <output data-testid="where">{`${pathname}${search}`}</output>;
}

/** AppShell 과 같은 연결: 훅의 sections·current·label 을 AppFrame 에, onClickCapture 를 감싼 곳에. */
function Frame() {
  const tabs = useSectionTabs();
  return (
    <div onClickCapture={tabs.onClickCapture}>
      <AppFrame brand={{ name: "punisher", href: "/" }} sections={tabs.sections} current={tabs.current} sectionsLabel={tabs.label}>
        <Where />
      </AppFrame>
    </div>
  );
}

function renderAt(path: string) {
  return render(
    <MemoryRouter initialEntries={[path]}>
      <UIProvider linkAs={Link}>
        <Frame />
      </UIProvider>
    </MemoryRouter>,
  );
}

const nav = () => screen.getByRole("navigation", { name: "punisher 섹션" });
const tab = (name: RegExp | string) => within(nav()).getByRole("link", { name });
const where = () => screen.getByTestId("where").textContent;

describe("useSectionTabs (standalone AppFrame sections)", () => {
  beforeEach(() => {
    state.restToday = false;
    resetTabMemory();
  });

  it("gives 오늘·지도·기록 with 복습 + 확인 대기 + 카드 next to 오늘", () => {
    renderAt("/map");
    const links = within(nav()).getAllByRole("link");
    expect(links.map((a) => a.textContent?.replace(/\d+/g, ""))).toEqual(["오늘", "지도", "기록"]);
    expect(tab("지도")).toHaveAttribute("aria-current", "page");
    // 2(복습) + 1(카드) + 2(확인 대기)
    expect(within(tab(/오늘/)).getByText("5")).toBeInTheDocument();
  });

  it("marks 오늘 on screens that belong to it (lecture, review session …)", () => {
    renderAt("/lectures/abc");
    expect(tab(/오늘/)).toHaveAttribute("aria-current", "page");
  });

  it("stays visible off the sections, with nothing marked current", () => {
    renderAt("/settings");
    expect(within(nav()).getAllByRole("link")).toHaveLength(3);
    for (const link of within(nav()).getAllByRole("link")) { expect(link).not.toHaveAttribute("aria-current"); }
  });

  it("shows no number on a rest day", () => {
    state.restToday = true;
    renderAt("/");
    expect(tab("오늘")).toBeInTheDocument();
  });

  it("returns to a section's last address, and to its first screen when it is already current", () => {
    renderAt("/map?panel=goals");
    fireEvent.click(tab("기록"));
    expect(where()).toBe("/history");
    expect(tab("지도")).toHaveAttribute("href", "/map?panel=goals");
    fireEvent.click(tab("지도"));
    expect(where()).toBe("/map?panel=goals");
    fireEvent.click(tab("지도"));
    expect(where()).toBe("/map");
  });

  it("leaves the brand link alone (it goes to the first screen, not 오늘's memory)", () => {
    renderAt("/lectures/abc");
    fireEvent.click(tab("지도"));
    expect(tab(/오늘/)).toHaveAttribute("href", "/lectures/abc");
    fireEvent.click(screen.getByRole("link", { name: "punisher 처음 화면" }));
    expect(where()).toBe("/");
  });
});
