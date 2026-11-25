import { describe, expect, it } from "vitest";
import { groupHits, hitCounts, hitHref, snippetParts } from "../lib/hits";

describe("search hits", () => {
  it("links lectures to the section and inquiries to their screen", () => {
    expect(hitHref({ kind: "lecture", refId: "l1", lectureId: "l1", anchor: "intro", title: "", sectionTitle: "", snippet: "" })).toBe("/lectures/l1?at=intro");
    expect(hitHref({ kind: "inquiry", refId: "q1", lectureId: "", anchor: "", title: "", sectionTitle: "", snippet: "" })).toBe("/inquiries/q1");
  });
  it("splits server marks or highlights the query", () => {
    expect(snippetParts("…커널은 [[디스크 블록]]을…", "디스크 블록")).toEqual([
      { text: "…커널은 ", match: false }, { text: "디스크 블록", match: true }, { text: "을…", match: false },
    ]);
    expect(snippetParts("캐시와 캐시", "캐시")).toEqual([
      { text: "캐시", match: true }, { text: "와 ", match: false }, { text: "캐시", match: true },
    ]);
  });
});

describe("search groups", () => {
  const lecture = (id: string, anchor: string) => ({ kind: "lecture" as const, refId: id, lectureId: id, anchor, title: `강의 ${id}`, sectionTitle: anchor, snippet: "" });
  const note = { kind: "note" as const, refId: "n1", lectureId: "a", anchor: "x", title: "메모", sectionTitle: "", snippet: "" };
  it("groups sections of one lecture under one title, keeping first-seen order", () => {
    const groups = groupHits([lecture("a", "s1"), note, lecture("b", "s1"), lecture("a", "s2"), lecture("a", "s3")]);
    expect(groups.map((g) => (g.type === "lecture" ? `${g.lectureId}:${g.hits.length}` : g.hit.kind))).toEqual(["a:3", "note", "b:1"]);
    expect(hitCounts(groups)).toEqual({ lectures: 2, places: 4, others: 1 });
  });
});
