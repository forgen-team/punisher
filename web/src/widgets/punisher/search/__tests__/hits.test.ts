import { describe, expect, it } from "vitest";
import { hitHref, snippetParts } from "../lib/hits";

describe("search hits", () => {
  it("links lectures to the section and inquiries to their screen", () => {
    expect(hitHref({ kind: "lecture", refId: "l1", lectureId: "l1", anchor: "intro", title: "", snippet: "" })).toBe("/lectures/l1?at=intro");
    expect(hitHref({ kind: "inquiry", refId: "q1", lectureId: "", anchor: "", title: "", snippet: "" })).toBe("/inquiries/q1");
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
