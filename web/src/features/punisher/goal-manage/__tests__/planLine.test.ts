import { describe, expect, it } from "vitest";
import { translate } from "@/src/shared/i18n";
import { planLine } from "../GoalDeadline";
import { messages } from "../i18n";

const base = { dueOn: "2026-12-10", remainingConcepts: 6, remainingMinutes: 120, studyDays: 10, minutesPerDay: 12, dailyMinutes: 30, finishOn: "2026-11-28" };

describe("goal plan line", () => {
  const t = translate(messages, "ko");
  it("says how much a day the deadline needs, without pressure words", () => {
    expect(planLine(t, { ...base, status: "on_track" }, "ko")).toContain("하루 약 12분");
    expect(planLine(t, { ...base, status: "tight", minutesPerDay: 45 }, "ko")).toContain("하루 약 45분이 필요해요");
    expect(planLine(t, { ...base, status: "no_deadline" }, "ko")).toContain("남은 개념 6개");
    expect(planLine(t, { ...base, status: "done" }, "ko")).toBe("경로를 다 마쳤어요.");
  });
});
