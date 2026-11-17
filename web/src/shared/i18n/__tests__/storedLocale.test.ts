import { afterEach, describe, expect, it, vi } from "vitest";
import { LOCALE_STORAGE_KEY, initialLocale, rememberLocale } from "../locale";

/** 단독판: 마지막 언어를 기억해 첫 화면을 그 언어로 그린다(깜박임 방지). */
describe("stored locale", () => {
  afterEach(() => {
    window.localStorage.clear();
    vi.restoreAllMocks();
  });

  it("remembers the last locale for the next start", () => {
    expect(initialLocale()).toBe("ko");
    rememberLocale("en");
    expect(window.localStorage.getItem(LOCALE_STORAGE_KEY)).toBe("en");
    expect(initialLocale()).toBe("en");
  });

  it("ignores unknown values", () => {
    window.localStorage.setItem(LOCALE_STORAGE_KEY, "fr");
    expect(initialLocale()).toBe("ko");
  });

  it("works without storage", () => {
    vi.spyOn(Storage.prototype, "getItem").mockImplementation(() => { throw new Error("blocked"); });
    vi.spyOn(Storage.prototype, "setItem").mockImplementation(() => { throw new Error("blocked"); });
    expect(() => rememberLocale("en")).not.toThrow();
    expect(initialLocale()).toBe("ko");
  });
});
