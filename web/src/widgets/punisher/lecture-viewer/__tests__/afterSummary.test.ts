import { describe, expect, it } from "vitest";
import type { LectureBlock } from "@/src/shared/api/punisher";
import { layoutAroundSummary } from "../lib/afterSummary";

const summary: LectureBlock = { anchor: "judgment_next-3", type: "summary", text: "- a\n- b\n- c" };

describe("요약 뒤 다음 개념", () => {
  it("moves the next-concepts paragraphs of an older lecture after the summary", () => {
    const md: LectureBlock = { anchor: "judgment_next-2", type: "markdown", text: "운영 신호는 409 비율입니다.\n\n다음으로 배울 개념입니다.\n\n- [조인](term:db.joins): 합친다." };
    const layout = layoutAroundSummary([{ anchor: "judgment_next-1", type: "markdown", text: "표" }, md, summary])!;
    expect(layout.before.map((b) => b.text)).toEqual(["표", "운영 신호는 409 비율입니다."]);
    expect(layout.before[1].anchor).toBe("judgment_next-2");
    expect(layout.after).toHaveLength(1);
    expect(layout.after[0].text).toContain("[조인](term:db.joins)");
    expect(layout.after[0].anchor).toBe("");
  });

  it("keeps a new lecture's order (next concepts already after the summary)", () => {
    const next: LectureBlock = { anchor: "judgment_next-4", type: "markdown", text: "다음에 배울 개념: [조인](term:db.joins)" };
    const layout = layoutAroundSummary([{ anchor: "judgment_next-1", type: "markdown", text: "판단 기준" }, summary, next])!;
    expect(layout.before).toHaveLength(1);
    expect(layout.after).toEqual([next]);
  });

  it("moves the whole block when it starts with the heading, and leaves lectures without a summary alone", () => {
    const md: LectureBlock = { anchor: "judgment_next-2", type: "markdown", text: "## What to learn next\n\n- [Joins](term:db.joins)" };
    expect(layoutAroundSummary([md, summary])!.after).toEqual([md]);
    expect(layoutAroundSummary([md])).toBeNull();
  });
});
