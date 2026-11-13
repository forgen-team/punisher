import { describe, expect, it } from "vitest";
import type { RecheckCard, Recommendation } from "@/src/shared/api/punisher";
import { recheckAction, todayRows } from "../lib/recheck";

function rec(kind: Recommendation["kind"], conceptId: string): Recommendation {
  return { kind, conceptId, conceptName: conceptId, area: "os", depth: 1, title: "", reason: "", practicalSituation: "", estimatedMinutes: 30 };
}

function card(conceptId: string, patch: Partial<RecheckCard> = {}): RecheckCard {
  return { conceptId, conceptName: conceptId, area: "os", goalId: "g1", due: false, ...patch };
}

const order = (rows: ReturnType<typeof todayRows>) => rows.map((r) => (r.type === "recheck" ? `re:${r.card.conceptId}` : r.item.conceptId));

describe("todayRows", () => {
  it("puts the recheck cards right before the first goal path recommendation", () => {
    const rows = todayRows([rec("study_request", "a"), rec("goal_path", "b"), rec("goal_path", "c")], [card("x"), card("y")]);
    expect(order(rows)).toEqual(["a", "re:x", "re:y", "b", "c"]);
  });

  it("puts them first without a goal path, and changes nothing without cards", () => {
    expect(order(todayRows([rec("essential_gap", "a")], [card("x")]))).toEqual(["re:x", "a"]);
    expect(order(todayRows([], [card("x")]))).toEqual(["re:x"]);
    expect(order(todayRows([rec("goal_path", "b")], undefined))).toEqual(["b"]);
  });
});

describe("recheckAction", () => {
  it("opens today's review when due, else the lecture's check, else the map", () => {
    expect(recheckAction(card("x", { due: true, lectureId: "l1" }))).toBe("review");
    expect(recheckAction(card("x", { lectureId: "l1" }))).toBe("check");
    expect(recheckAction(card("x", { lectureId: "" }))).toBe("map");
  });
});
