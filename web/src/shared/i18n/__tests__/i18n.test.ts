import { describe, expect, it } from "vitest";
import { defineMessages, translate } from "../messages";

describe("i18n", () => {
  const m = defineMessages({
    ko: { title: "오늘", count: (n: number) => `${n}개` },
    en: { title: "Today", count: (n: number) => `${n} items` },
  });
  it("picks the locale's catalog", () => {
    expect(translate(m, "ko").title).toBe("오늘");
    expect(translate(m, "en").count(3)).toBe("3 items");
  });
});
