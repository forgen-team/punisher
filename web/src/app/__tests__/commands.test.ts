import { describe, expect, it, vi } from "vitest";
import { translate } from "@/src/shared/i18n";
import { messages } from "@/src/app/i18n";
import type { Recommendation } from "@/src/shared/api/punisher";
import { buildCommands, nextLecture } from "../commands";

const rec = (over: Partial<Recommendation>): Recommendation => ({
  kind: "goal_path", conceptId: "c1", conceptName: "TCP", area: "net", depth: 0, title: "TCP 연결", reason: "", practicalSituation: "", estimatedMinutes: 20,
  ...over,
});

describe("shell commands", () => {
  const t = translate(messages, "ko");
  it("registers the §3 list — unique, valid, within the contract limits", () => {
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
    expect(list.map((c) => c.label.startsWith("다음 강의:") ? "다음 강의" : c.label)).toEqual([
      "punisher 오늘", "다음 강의", "지도", "기록", "복습 시작", "카드로 복습", "기록에서 찾기", "새 챕터", "목표 관리", "강의 프롬프트", "punisher 설정",
    ]);
    expect(list.find((c) => c.id === "find")?.href).toBe("/history");
    expect(list.find((c) => c.id === "goals")?.href).toBe("/map?panel=goals");
    expect(list.some((c) => c.href?.startsWith("/search"))).toBe(false);
    list.find((c) => c.id === "review")?.onSelect?.();
    expect(start).toHaveBeenCalledTimes(1);
  });
  it("leaves out the next lecture when there is none", () => {
    expect(buildCommands(t, { startReview: () => undefined }).some((c) => c.id === "next-lecture")).toBe(false);
  });
  it("next lecture opens the lecture, or starts one from the concept", () => {
    expect(nextLecture([rec({ kind: "resume", lectureId: "r1" }), rec({ lectureId: "l9" })])).toEqual({ title: "TCP", href: "/lectures/l9" });
    expect(nextLecture([rec({ conceptId: "c 2" })])?.href).toBe("/start?concept=c+2");
    expect(nextLecture([])).toBeUndefined();
  });
});
