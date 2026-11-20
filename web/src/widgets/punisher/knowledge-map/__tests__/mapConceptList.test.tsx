import { fireEvent, render, screen, within } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { GraphNode } from "@/src/shared/api/punisher";
import { visibleFoundations } from "../lib/foundationsView";
import { MapConceptList } from "../ui/MapConceptList";

const node = (over: Partial<GraphNode> & Pick<GraphNode, "id" | "name" | "depth">): GraphNode => ({
  area: "os", status: "active", state: "not_started", locked: false, frontier: false, unlocks: 0, ...over,
});

const nodes: GraphNode[] = [
  node({ id: "os.paging", name: "페이징", depth: 2, locked: true }),
  node({ id: "found.var", name: "변수", depth: 0, area: "foundations", hiddenByDefault: true, state: "completed" }),
  node({ id: "os.process", name: "프로세스", depth: 1, state: "in_progress" }),
  node({ id: "os.thread", name: "스레드", depth: 1, frontier: true }),
  node({ id: "os.numa", name: "NUMA", depth: 3, dismissed: true }),
];

function layers() {
  return screen.getAllByRole("heading").map((h) => h.textContent);
}

describe("지도 목록 보기", () => {
  it("groups concepts by depth layer in order, with the node state as row meta", () => {
    render(<MapConceptList nodes={visibleFoundations(nodes, true, null)} selectedId={null} showArea={false} onSelect={vi.fn()} />);
    expect(layers()).toEqual(["0단계 기초", "깊이 1 · 기본", "깊이 2 · 응용", "깊이 3 · 심화"]);
    const basics = screen.getByRole("region", { name: "깊이 1 · 기본" });
    expect(within(basics).getAllByRole("button").map((b) => b.textContent)).toEqual(["프로세스진행 중", "스레드추천(다음 학습)"]);
    expect(screen.getByRole("button", { name: /페이징/ })).toHaveTextContent("잠김(선행 미완)");
    expect(screen.getByRole("button", { name: /NUMA/ })).toHaveTextContent("필요 없음 · 미학습");
  });

  it("leaves out the foundations layer while foundations are hidden", () => {
    render(<MapConceptList nodes={visibleFoundations(nodes, false, null)} selectedId={null} showArea={false} onSelect={vi.fn()} />);
    expect(layers()).toEqual(["깊이 1 · 기본", "깊이 2 · 응용", "깊이 3 · 심화"]);
    expect(screen.queryByText("변수")).toBeNull();
  });

  it("opens the concept detail through onSelect when a row is tapped", () => {
    const onSelect = vi.fn();
    render(<MapConceptList nodes={nodes} selectedId="os.thread" showArea onSelect={onSelect} />);
    fireEvent.click(screen.getByRole("button", { name: /프로세스/ }));
    expect(onSelect).toHaveBeenCalledWith("os.process");
    expect(screen.getByRole("button", { name: /스레드/ })).toHaveAttribute("aria-current", "true");
  });
});
