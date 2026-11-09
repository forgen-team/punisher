import { describe, expect, it } from "vitest";
import { changedLines, findUnknownVariables, lineDiff, withSuggestion } from "../lib/lineDiff";

describe("lineDiff", () => {
  it("marks unchanged text as same only", () => {
    const d = lineDiff("a\nb", "a\nb");
    expect(d.every((l) => l.kind === "same")).toBe(true);
    expect(changedLines(d)).toBe(0);
  });

  it("finds an inserted and a removed line in order", () => {
    const d = lineDiff("원칙1\n원칙2\n원칙3", "원칙1\n새 원칙\n원칙3\n추가");
    expect(d).toEqual([
      { kind: "same", text: "원칙1" },
      { kind: "del", text: "원칙2" },
      { kind: "add", text: "새 원칙" },
      { kind: "same", text: "원칙3" },
      { kind: "add", text: "추가" },
    ]);
    expect(changedLines(d)).toBe(3);
  });
});

describe("findUnknownVariables", () => {
  it("lists undeclared placeholders once", () => {
    expect(findUnknownVariables("{{concept_name}} {{ foo }} {{foo}} {{bar}}", ["concept_name"])).toEqual(["foo", "bar"]);
  });
});

describe("withSuggestion", () => {
  it("appends the feedback suggestion once", () => {
    const once = withSuggestion("본문\n", "단계를 더 잘게");
    expect(once).toBe("본문\n\n## 내 피드백 반영\n단계를 더 잘게");
    expect(withSuggestion(once, "단계를 더 잘게")).toBe(once);
  });
});
