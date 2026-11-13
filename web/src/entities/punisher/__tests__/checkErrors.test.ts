import { describe, expect, it } from "vitest";
import { PunisherApiError, describePunisherError } from "@/src/shared/api/punisher";
import { describeCheckError } from "../model/checks";
import { chunkIds } from "../model/reviews";
import { CHECK_ERROR_MESSAGES } from "../ui/constants";

const err = (status: number, code: string, message = "server") => new PunisherApiError(status, code, message);

describe("describeCheckError", () => {
  it("maps 400 by code, not by message text", () => {
    expect(describeCheckError(err(400, "ANSWER_EMPTY", "objectives reason"))).toBe(CHECK_ERROR_MESSAGES.ANSWER_EMPTY);
    expect(describeCheckError(err(400, "ANSWER_TOO_LONG"))).toBe(CHECK_ERROR_MESSAGES.ANSWER_TOO_LONG);
    expect(describeCheckError(err(400, "NO_OBJECTIVES"))).toBe(CHECK_ERROR_MESSAGES.NO_OBJECTIVES);
    expect(describeCheckError(err(400, "INVALID_VERDICT"))).toBe(CHECK_ERROR_MESSAGES.INVALID_VERDICT);
    expect(describeCheckError(err(400, "BAD_REQUEST", "invalid JSON body"))).toBe(CHECK_ERROR_MESSAGES.BAD_REQUEST);
  });

  it("keeps the server's 409 text and never the shared lecture-generation 409 text", () => {
    const busy = err(409, "CONFLICT", "채점 중이에요. 끝난 뒤 다시 시도해 주세요");
    expect(describeCheckError(busy)).toBe("채점 중이에요. 끝난 뒤 다시 시도해 주세요");
    expect(describeCheckError(busy)).not.toBe(describePunisherError(busy));
    expect(describeCheckError(err(409, "NO_REVIEWS", "오늘 복습할 항목이 없어요: 복습 시간이 0분이에요"))).toBe(CHECK_ERROR_MESSAGES.NO_REVIEWS);
  });

  it("falls back to the shared text for other statuses", () => {
    expect(describeCheckError(err(404, "NOT_FOUND"))).toBe(describePunisherError(err(404, "NOT_FOUND")));
  });
});

describe("chunkIds", () => {
  it("splits by the batch limit", () => {
    const ids = Array.from({ length: 601 }, (_, index) => `c${index}`);
    expect(chunkIds(ids).map((c) => c.length)).toEqual([300, 300, 1]);
    expect(chunkIds([])).toEqual([]);
  });
});
