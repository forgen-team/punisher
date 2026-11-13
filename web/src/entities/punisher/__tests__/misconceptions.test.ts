import { describe, expect, it } from "vitest";
import type { CheckDetail } from "@/src/shared/api/punisher";
import { withAcknowledged } from "../model/misconceptions";

describe("withAcknowledged", () => {
  const base = { misconceptionTurnIds: ["t1"] } as unknown as CheckDetail;
  it("adds the turn once and keeps the same object when already there", () => {
    expect(withAcknowledged(base, "t2").misconceptionTurnIds).toEqual(["t1", "t2"]);
    expect(withAcknowledged(base, "t1")).toBe(base);
    expect(withAcknowledged({} as CheckDetail, "t9").misconceptionTurnIds).toEqual(["t9"]);
  });
});
