import { describe, expect, it } from "vitest";
import { sessionTimeLabel } from "../lib/sessionTime";

describe("sessionTimeLabel", () => {
  it("joins lecture, check and review minutes", () => {
    expect(sessionTimeLabel({ lectureMinutes: 20, checkMinutes: 7, reviewMinutes: 5 }, "ko")).toBe("강의 약 20분 + 확인 약 7분 + 복습 약 5분");
    expect(sessionTimeLabel({ lectureMinutes: 20, checkMinutes: 0, reviewMinutes: 5 }, "en")).toBe("Lecture about 20 min + Review about 5 min");
  });
  it("drops zero parts and rounds", () => {
    expect(sessionTimeLabel({ lectureMinutes: 19.6, checkMinutes: 6.8, reviewMinutes: 0 }, "ko")).toBe("강의 약 20분 + 확인 약 7분");
  });
  it("is empty without a session or with all zero", () => {
    expect(sessionTimeLabel(undefined, "ko")).toBe("");
    expect(sessionTimeLabel({ lectureMinutes: 0, checkMinutes: 0, reviewMinutes: 0 }, "ko")).toBe("");
  });
});
