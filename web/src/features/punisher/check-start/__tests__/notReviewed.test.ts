import { describe, expect, it } from "vitest";
import { PunisherApiError } from "@/src/shared/api/punisher";
import { isLectureNotReviewed } from "../useLaunchCheck";

describe("isLectureNotReviewed", () => {
  it("matches only 409 LECTURE_NOT_REVIEWED", () => {
    expect(isLectureNotReviewed(new PunisherApiError(409, "LECTURE_NOT_REVIEWED", "x"))).toBe(true);
    expect(isLectureNotReviewed(new PunisherApiError(409, "NO_OBJECTIVES", "x"))).toBe(false);
    expect(isLectureNotReviewed(new PunisherApiError(400, "LECTURE_NOT_REVIEWED", "x"))).toBe(false);
    expect(isLectureNotReviewed(new Error("x"))).toBe(false);
  });
});
