import { describe, expect, it } from "vitest";
import { FAMILIARITY_AREAS, PUNISHER_AREAS, type GraphNode } from "@/src/shared/api/punisher";
import { hiddenFoundationsCount, visibleFoundations } from "../lib/foundationsView";

const node = (id: string, area: string, hiddenByDefault?: boolean) => ({ id, area, hiddenByDefault }) as unknown as GraphNode;

describe("foundationsView", () => {
  const nodes = [node("found.a", "foundations", true), node("found.b", "foundations", true), node("net.http", "network-web")];

  it("hides default-hidden foundations until expanded", () => {
    expect(visibleFoundations(nodes, false, null).map((n) => n.id)).toEqual(["net.http"]);
    expect(visibleFoundations(nodes, true, null)).toHaveLength(3);
    expect(hiddenFoundationsCount(nodes)).toBe(2);
  });

  it("keeps foundations when the area is picked or the level shows them", () => {
    expect(visibleFoundations(nodes.slice(0, 2), false, "foundations")).toHaveLength(2);
    const shown = [node("found.a", "foundations"), node("net.http", "network-web")];
    expect(visibleFoundations(shown, false, null)).toHaveLength(2);
    expect(hiddenFoundationsCount(shown)).toBe(0);
  });

  it("first setup asks the 8 areas without foundations", () => {
    expect(PUNISHER_AREAS).toContain("foundations");
    expect(FAMILIARITY_AREAS).toHaveLength(8);
    expect(FAMILIARITY_AREAS as readonly string[]).not.toContain("foundations");
  });
});
