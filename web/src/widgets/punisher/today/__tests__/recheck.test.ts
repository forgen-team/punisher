import { describe, expect, it } from "vitest";
import type { RecheckCard, Recommendation, UncheckedCard } from "@/src/shared/api/punisher";
import { recheckAction, todayRows } from "../lib/recheck";

function rec(kind: Recommendation["kind"], conceptId: string): Recommendation {
  return { kind, conceptId, conceptName: conceptId, area: "os", depth: 1, title: "", reason: "", practicalSituation: "", estimatedMinutes: 30 };
}

function card(conceptId: string, patch: Partial<RecheckCard> = {}): RecheckCard {
  return { conceptId, conceptName: conceptId, area: "os", goalId: "g1", due: false, ...patch };
}

const PREFIX = { recheck: "re:", unchecked: "un:" } as const;

const order = (rows: ReturnType<typeof todayRows>) =>
  rows.map((r) => {
    if (r.type === "synthesis") { return "synthesis"; }
    return r.type === "recommendation" ? r.item.conceptId : `${PREFIX[r.type]}${r.card.conceptId}`;
  });

function unchecked(conceptId: string): UncheckedCard {
  return { lectureId: `l-${conceptId}`, conceptId, conceptName: conceptId, area: "os", title: "" };
}

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

  it("puts the unchecked lectures' checks at the very front", () => {
    expect(order(todayRows([rec("goal_path", "b")], [card("x")], [unchecked("u")]))).toEqual(["un:u", "re:x", "b"]);
    expect(order(todayRows([rec("study_request", "a"), rec("goal_path", "b")], [card("x")], [unchecked("u")]))).toEqual(["un:u", "a", "re:x", "b"]);
    expect(order(todayRows([rec("essential_gap", "a")], undefined, [unchecked("u"), unchecked("v")]))).toEqual(["un:u", "un:v", "a"]);
  });
});

describe("recheckAction", () => {
  it("opens today's review when due, else the lecture's check, else the map", () => {
    expect(recheckAction(card("x", { due: true, lectureId: "l1" }))).toBe("review");
    expect(recheckAction(card("x", { lectureId: "l1" }))).toBe("check");
    expect(recheckAction(card("x", { lectureId: "" }))).toBe("map");
  });
  it("puts this week's synthesis first, before the unchecked lectures", () => {
    const rows = todayRows([rec("goal_path", "b")], [], [unchecked("u")], true);
    expect(order(rows)).toEqual(["synthesis", "un:u", "b"]);
  });
});
