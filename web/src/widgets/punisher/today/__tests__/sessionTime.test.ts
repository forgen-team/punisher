import { describe, expect, it } from "vitest";
import { sessionTimeLabel } from "../lib/sessionTime";

describe("sessionTimeLabel", () => {
  it("labels the total and its parts", () => {
    expect(sessionTimeLabel({ lectureMinutes: 20, checkMinutes: 7, reviewMinutes: 2 }, "ko")).toBe("오늘 할 일 약 29분: 강의 20 · 확인 7 · 복습 2");
    expect(sessionTimeLabel({ lectureMinutes: 20, checkMinutes: 0, reviewMinutes: 5 }, "en")).toBe("Today about 25 min: lecture 20 · review 5");
  });
  it("drops zero parts and rounds each part before summing", () => {
    expect(sessionTimeLabel({ lectureMinutes: 19.6, checkMinutes: 6.8, reviewMinutes: 0 }, "ko")).toBe("오늘 할 일 약 27분: 강의 20 · 확인 7");
    expect(sessionTimeLabel({ lectureMinutes: 0, checkMinutes: 0.3, reviewMinutes: 0 }, "ko")).toBe("오늘 할 일 약 1분: 확인 1");
  });
  it("is empty without a session or with all zero", () => {
    expect(sessionTimeLabel(undefined, "ko")).toBe("");
    expect(sessionTimeLabel({ lectureMinutes: 0, checkMinutes: 0, reviewMinutes: 0 }, "ko")).toBe("");
  });
});
