import { describe, expect, it, vi } from "vitest";
import { translate } from "@/src/shared/i18n";
import { messages } from "@/src/app/i18n";
import { buildCommands } from "../commands";

describe("shell commands", () => {
  const t = translate(messages, "ko");
  it("registers unique, valid commands within the contract limits", () => {
    const start = vi.fn();
    const list = buildCommands(t, { next: { title: "TCP 연결 수립을 아주 길게 설명하는 강의 제목입니다 정말로 길어요 계속 길어요", href: "/lectures/l1" }, startReview: start });
    expect(new Set(list.map((c) => c.id)).size).toBe(list.length);
    expect(list.length).toBeLessThanOrEqual(50);
    for (const c of list) {
      expect(c.label.length).toBeLessThanOrEqual(60);
      expect(Boolean(c.href) || Boolean(c.onSelect)).toBe(true);
      expect(c.href === undefined || c.href.startsWith("/")).toBe(true);
      expect((c.keywords ?? []).length).toBeLessThanOrEqual(10);
    }
    list.find((c) => c.id === "review")?.onSelect?.();
    expect(start).toHaveBeenCalledTimes(1);
    expect(list[1]?.id).toBe("next-lecture");
  });
  it("leaves out the next lecture when there is none", () => {
    expect(buildCommands(t, { startReview: () => undefined }).some((c) => c.id === "next-lecture")).toBe(false);
  });
});
