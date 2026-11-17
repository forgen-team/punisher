import { describe, expect, it } from "vitest";
import { resumePosition } from "../resumeProgress";

describe("resumePosition", () => {
  it("names the part from the saved anchor", () => {
    expect(resumePosition("principles-1", "ko")).toBe("2/4부분");
    expect(resumePosition("judgment_next", "ko")).toBe("4/4부분");
    expect(resumePosition("principles-1", "en")).toBe("Part 2/4");
  });

  it("is empty when the position is unknown", () => {
    expect(resumePosition("", "ko")).toBe("");
    expect(resumePosition(undefined, "ko")).toBe("");
    expect(resumePosition("summary-2", "ko")).toBe("");
  });
});
