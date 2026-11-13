import { describe, expect, it } from "vitest";
import { PunisherApiError, describePunisherError } from "@/src/shared/api/punisher";
import { describeSettingsError } from "../model/reading";
import { SETTINGS_ERROR_MESSAGES } from "../ui/constants";

const err = (status: number, code: string, message = "server") => new PunisherApiError(status, code, message);

describe("describeSettingsError", () => {
  it("maps a 400 by code and never shows the server's English text", () => {
    expect(describeSettingsError(err(400, "INVALID_INTERVALS", "reviewIntervals must be 1~6 increasing positive day counts"))).toBe(
      SETTINGS_ERROR_MESSAGES.INVALID_INTERVALS,
    );
    expect(describeSettingsError(err(400, "INVALID_DAILY_MINUTES"))).toBe(SETTINGS_ERROR_MESSAGES.INVALID_DAILY_MINUTES);
    expect(describeSettingsError(err(400, "BAD_REQUEST", "invalid JSON body"))).toBe(SETTINGS_ERROR_MESSAGES.BAD_REQUEST);
    expect(describeSettingsError(err(400, "SOMETHING_NEW", "invalid thing"))).toBe(SETTINGS_ERROR_MESSAGES.BAD_REQUEST);
  });

  it("falls back to the shared text for other statuses", () => {
    expect(describeSettingsError(err(500, "INTERNAL_ERROR"))).toBe(describePunisherError(err(500, "INTERNAL_ERROR")));
  });
});
