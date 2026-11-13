import { describe, expect, it } from "vitest";
import type { LectureBlock } from "@/src/shared/api/punisher";
import { groupBlocks, initialReveal, revealNext } from "../lib/blockGroups";

const b = (anchor: string, type: LectureBlock["type"]): LectureBlock => ({ anchor, type, text: anchor });

describe("groupBlocks", () => {
  it("folds the output right after a pause", () => {
    const groups = groupBlocks([b("p", "pause"), b("o", "log"), b("m", "markdown")]);
    expect(groups.map((g) => g.type)).toEqual(["pause", "single"]);
    expect(groups[0].type === "pause" && groups[0].output?.anchor).toBe("o");
  });
  it("leaves a pause alone when the next block is not output", () => {
    const groups = groupBlocks([b("p", "pause"), b("m", "markdown")]);
    expect(groups[0].type === "pause" && groups[0].output).toBeUndefined();
    expect(groups).toHaveLength(2);
  });
  it("pairs exercise with the next answer", () => {
    const groups = groupBlocks([b("e", "exercise"), b("a", "answer"), b("a2", "answer")]);
    expect(groups.map((g) => g.type)).toEqual(["exercise", "single"]);
    expect(groups[0].type === "exercise" && groups[0].answer?.anchor).toBe("a");
  });
});

describe("trace reveal", () => {
  it("starts with one row and grows to the total", () => {
    expect(initialReveal(5)).toBe(1);
    expect(initialReveal(0)).toBe(0);
    expect(revealNext(1, 3)).toBe(2);
    expect(revealNext(3, 3)).toBe(3);
  });
});
