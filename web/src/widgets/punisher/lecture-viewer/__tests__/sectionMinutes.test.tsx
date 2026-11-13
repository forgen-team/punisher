import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import type { LectureSection } from "@/src/shared/api/punisher";
import { sectionMinutes } from "../lib/sectionMinutes";
import { LectureBody } from "../ui/LectureBody";

describe("섹션 분", () => {
  it("prefers the server estimate over the model's minutes", () => {
    expect(sectionMinutes({ minutes: 9, estimatedMinutes: 6 })).toBe(6);
    expect(sectionMinutes({ minutes: 9 })).toBe(9);
    const section: LectureSection = { anchor: "s1", kind: "principles", title: "원리", minutes: 9, estimatedMinutes: 6, blocks: [] };
    render(<LectureBody content={{ sections: [section] }} />);
    expect(screen.getByText(/약 6분/)).toBeInTheDocument();
    expect(screen.queryByText(/약 9분/)).toBeNull();
  });
});
