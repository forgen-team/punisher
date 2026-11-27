import { fireEvent, render, screen, within } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { describe, expect, it, vi } from "vitest";
import type { AreaCoverage, GraphNode } from "@/src/shared/api/punisher";
import { MapConceptList, groupByArea } from "../ui/MapConceptList";

const node = (over: Partial<GraphNode> & Pick<GraphNode, "id" | "name" | "depth">): GraphNode => ({
  area: "os", status: "active", state: "not_started", locked: false, frontier: false, unlocks: 0, ...over,
});

const nodes: GraphNode[] = [
  node({ id: "db.index", name: "인덱스", depth: 1, area: "database", state: "completed" }),
  node({ id: "os.paging", name: "페이징", depth: 2, locked: true }),
  node({ id: "os.process", name: "프로세스", depth: 1, state: "in_progress" }),
  node({ id: "os.thread", name: "스레드", depth: 1, frontier: true }),
  node({ id: "os.numa", name: "NUMA", depth: 3, dismissed: true }),
  node({ id: "os.cfs", name: "CFS", depth: 2, requests: [{ id: "r1", query: "CFS", source: "manual", status: "unresolved" }] }),
  node({ id: "os.sched", name: "스케줄러", depth: 1, state: "known" }),
  node({ id: "other.x", name: "기타 개념", depth: 1, area: "other" }),
];

const coverage: AreaCoverage[] = [
  { area: "os", total: 8, completed: 1, known: 2, inProgress: 1, requested: 1, dismissed: 0 },
  { area: "database", total: 4, completed: 1, known: 0, inProgress: 0, requested: 0, dismissed: 0 },
];

function renderList(over: Partial<React.ComponentProps<typeof MapConceptList>> = {}) {
  return render(
    <MemoryRouter>
      <MapConceptList nodes={nodes} coverage={coverage} selectedId={null} onSelect={vi.fn()} {...over} />
    </MemoryRouter>,
  );
}

describe("지도 목록 보기", () => {
  it("groups by area in the fixed area order and keeps unknown areas last", () => {
    expect(groupByArea(nodes).map((g) => g.area)).toEqual(["os", "database", "other"]);
  });

  it("orders concepts inside an area by depth", () => {
    const os = groupByArea(nodes).find((g) => g.area === "os");
    expect(os?.nodes.map((n) => n.name)).toEqual(["프로세스", "스레드", "스케줄러", "페이징", "CFS", "NUMA"]);
  });

  it("shows each area's progress percent in its heading and the state name as row meta", () => {
    renderList();
    expect(screen.getAllByRole("heading").map((h) => h.textContent)).toEqual(["운영체제", "데이터베이스", "기타"]);
    const os = screen.getByRole("heading", { name: "운영체제" }).closest("section") as HTMLElement;
    expect(within(os).getByText("38%")).toBeInTheDocument();
    expect(within(os).getByRole("button", { name: /프로세스/ })).toHaveTextContent("읽는 중");
    expect(within(os).getByRole("button", { name: /스케줄러/ })).toHaveTextContent("이미 앎");
    expect(within(os).getByRole("button", { name: /스레드/ })).toHaveTextContent("추천");
    expect(within(os).getByRole("button", { name: /페이징/ })).toHaveTextContent("잠김");
    expect(within(os).getByRole("button", { name: /CFS/ })).toHaveTextContent("요청됨");
    expect(within(os).getByRole("button", { name: /NUMA/ })).toHaveTextContent(/^.*필요 없음$/);
    const db = screen.getByRole("heading", { name: "데이터베이스" }).closest("section") as HTMLElement;
    expect(within(db).getByRole("button", { name: /인덱스/ })).toHaveTextContent("배움");
  });

  it("opens the concept through onSelect when a row is tapped, and marks the selected row", () => {
    const onSelect = vi.fn();
    renderList({ selectedId: "os.thread", onSelect });
    fireEvent.click(screen.getByRole("button", { name: /프로세스/ }));
    expect(onSelect).toHaveBeenCalledWith("os.process");
    // ui ListRow 는 선택 표시(aria-current)를 행(li)에 둔다.
    expect(screen.getByRole("button", { name: /스레드/ }).closest("li")).toHaveAttribute("aria-current", "true");
  });
});
