import { describe, expect, it } from "vitest";
import type { LectureSection } from "@/src/shared/api/punisher";
import { extractLectureTerms } from "../lib/lectureTerms";

function section(anchor: string, blocks: LectureSection["blocks"]): LectureSection {
  return { anchor, kind: "principles", title: anchor, minutes: 5, blocks } as LectureSection;
}

describe("extractLectureTerms", () => {
  it("collects term blocks and inline term links in order, once per concept", () => {
    const terms = extractLectureTerms({
      sections: [
        section("s1", [
          { type: "markdown", text: "[해시](term:ds.hash)와 [배열](term:ds.array)을 비교한다." },
          { type: "term", term: "해시 테이블", conceptId: "ds.hash", text: "" },
        ] as LectureSection["blocks"]),
        section("s2", [
          { type: "misconception", text: "@@term concept=os.thread term=스레드 는 가볍다" },
          { type: "code", text: "[x](term:not.a.term)" },
          { type: "keypoint", kind: "rule", text: "[캐시](term:arch.cache)는 가깝다" },
          { type: "exercise", text: "[스택](term:ds.stack)에 넣으면?" },
          { type: "answer", text: "[큐](term:ds.queue)와 다르다" },
          { type: "summary", text: "- [힙](term:ds.heap)" },
        ] as LectureSection["blocks"]),
      ],
    });
    expect(terms).toEqual([
      { conceptId: "ds.hash", term: "해시", sectionAnchor: "s1" },
      { conceptId: "ds.array", term: "배열", sectionAnchor: "s1" },
      { conceptId: "os.thread", term: "스레드", sectionAnchor: "s2" },
      { conceptId: "arch.cache", term: "캐시", sectionAnchor: "s2" },
      { conceptId: "ds.stack", term: "스택", sectionAnchor: "s2" },
      { conceptId: "ds.queue", term: "큐", sectionAnchor: "s2" },
      { conceptId: "ds.heap", term: "힙", sectionAnchor: "s2" },
    ]);
  });
});
