import { afterEach, describe, expect, it } from "vitest";
import { PunisherApiError, describePunisherError } from "@/src/shared/api/punisher";
import { setCurrentLocale } from "@/src/shared/i18n";
import { describeCheckError } from "../model/checks";
import { AREA_LABELS, CHECK_ERROR_MESSAGES, areaLabel, masteryLabel } from "../ui/constants";

afterEach(() => setCurrentLocale("ko"));

describe("labels follow the current locale", () => {
  it("keeps the same Record names and switches values", () => {
    expect(AREA_LABELS.os).toBe("운영체제");
    setCurrentLocale("en");
    expect(AREA_LABELS.os).toBe("Operating systems");
    expect(Object.keys(AREA_LABELS)).toContain("database");
    expect(areaLabel("other")).toBe("Other");
    expect(masteryLabel(5, "ko")).toBe("설명 가능");
    expect(describeCheckError(new PunisherApiError(400, "ANSWER_EMPTY", "x"))).toBe(CHECK_ERROR_MESSAGES.ANSWER_EMPTY);
    expect(CHECK_ERROR_MESSAGES.ANSWER_EMPTY).toBe("Write an answer.");
    expect(describePunisherError(new PunisherApiError(404, "NOT_FOUND", "x"))).toBe("Couldn't find the requested item.");
  });
});
