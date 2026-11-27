import { describe, expect, it } from "vitest";
import { START, graded, reveal } from "../lib/session";

describe("card session", () => {
  it("grades only after the answer is shown, then moves on", () => {
    expect(graded(START)).toBe(START);
    const shown = reveal(START);
    expect(shown.revealed).toBe(true);
    expect(reveal(shown)).toBe(shown);
    expect(graded(shown, 3)).toEqual({ index: 1, revealed: false, graded: 1, again: 0, hard: 0 });
  });

  it("counts 'again' and 'hard' separately", () => {
    expect(graded(reveal(START), 1)).toEqual({ index: 1, revealed: false, graded: 1, again: 1, hard: 0 });
    expect(graded(reveal(START), 2)).toEqual({ index: 1, revealed: false, graded: 1, again: 0, hard: 1 });
  });
});
