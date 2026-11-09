import { describe, expect, it } from "vitest";
import type { GraphEdge, GraphNode } from "@/src/shared/api/punisher";
import { chapterView } from "../lib/chapterView";

const node = (id: string, chapterId?: string) => ({ id, chapterId }) as unknown as GraphNode;

describe("chapterView", () => {
  it("keeps the chapter's concepts and their direct prerequisites only", () => {
    const nodes = [node("seed.http"), node("seed.other"), node("u1.a", "ch1"), node("u1.b", "ch1"), node("u2.x", "ch2")];
    const edges: GraphEdge[] = [
      { from: "seed.http", to: "u1.a", kind: "prerequisite" },
      { from: "u1.a", to: "u1.b", kind: "prerequisite" },
      { from: "seed.other", to: "u2.x", kind: "prerequisite" },
      { from: "seed.other", to: "u1.b", kind: "related" },
    ];
    expect(chapterView(nodes, edges, "ch1").map((n) => n.id)).toEqual(["seed.http", "u1.a", "u1.b"]);
  });
});
