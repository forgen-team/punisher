import { describe, expect, it } from "vitest";
import { resumePosition } from "../resumeProgress";

describe("resumePosition", () => {
  it("names the part from the saved anchor", () => {
    expect(resumePosition("principles-1")).toBe("2/4부분");
    expect(resumePosition("judgment_next")).toBe("4/4부분");
  });

  it("is empty when the position is unknown", () => {
    expect(resumePosition("")).toBe("");
    expect(resumePosition(undefined)).toBe("");
    expect(resumePosition("summary-2")).toBe("");
  });
});
