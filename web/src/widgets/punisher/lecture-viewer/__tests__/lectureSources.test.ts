import { describe, expect, it } from "vitest";
import type { LectureSection } from "@/src/shared/api/punisher";
import { sourcesNotInSections } from "../lib/lectureSources";

const link = (url: string) => ({ title: url, url });
const section = (blocks: LectureSection["blocks"]): LectureSection => ({ anchor: "s", kind: "principles", title: "원리", minutes: 0, blocks });

describe("sourcesNotInSections", () => {
  it("섹션 출처에 나온 링크는 빼고 나머지만 남긴다", () => {
    const content = { sections: [section([{ anchor: "a", type: "sources", sources: [link("u1"), link("u2")] }]), section([{ anchor: "b", type: "markdown", text: "본문" }])] };
    expect(sourcesNotInSections([link("u1"), link("u2"), link("u3")], content)).toEqual([link("u3")]);
    expect(sourcesNotInSections([link("u1")], content)).toEqual([]);
  });

  it("섹션 출처가 없으면 그대로", () => {
    expect(sourcesNotInSections([link("u1")], { sections: [section([])] })).toEqual([link("u1")]);
  });
});
