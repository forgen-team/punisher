import { renderHook } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { setWitemsRuntime } from "@/src/shared/config/runtime";
import { useSectionBadges } from "../sectionBadges";

vi.mock("@/src/entities/punisher", () => ({
  useActivity: () => ({ data: { restToday: false, today: { reviewsDue: 2, cardsDue: 1 } } }),
  useToday: () => ({ data: { items: [], exhausted: false, unchecked: [{ lectureId: "l1" }] } }),
}));

describe("useSectionBadges", () => {
  afterEach(() => setWitemsRuntime(null));

  it("sends 복습 + 확인 대기 + 카드 as the 오늘 section badge, keyed by its path", () => {
    const setBadges = vi.fn();
    setWitemsRuntime({ apiBase: "/apps-api/punisher/", portal: document.body, setTitle: () => undefined, setBadges });
    renderHook(() => useSectionBadges());
    expect(setBadges).toHaveBeenLastCalledWith({ "/": 4 });
  });

  it("does nothing in a shell without setBadges", () => {
    setWitemsRuntime({ apiBase: "/apps-api/punisher/", portal: document.body, setTitle: () => undefined });
    expect(() => renderHook(() => useSectionBadges())).not.toThrow();
  });
});
