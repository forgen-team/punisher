import { describe, expect, it } from "vitest";
import { translate } from "@/src/shared/i18n";
import { layoutActions, withAppSettings } from "../actionLayout";
import { backFallbackFor, pageWidthFor, usesGeneration } from "../frame";
import { messages } from "../i18n";
import { titleKeyFor } from "../titles";

const t = translate(messages, "ko");

describe("frame rules", () => {
  it("picks the page width from the path", () => {
    expect(pageWidthFor("/")).toBe("full");
    expect(pageWidthFor("/map")).toBe("full");
    expect(pageWidthFor("/lectures/1")).toBe("full");
    expect(pageWidthFor("/inquiries/1")).toBe("reading");
    expect(pageWidthFor("/settings")).toBe("narrow");
    expect(pageWidthFor("/settings/prompts")).toBe("wide");
    expect(pageWidthFor("/history")).toBe("default");
    expect(pageWidthFor("/check/1")).toBe("default");
  });

  it("← goes nowhere on section first screens and to the section first screen otherwise", () => {
    expect(backFallbackFor("/")).toBeNull();
    expect(backFallbackFor("/map")).toBeNull();
    expect(backFallbackFor("/history")).toBeNull();
    for (const path of ["/lectures/1", "/check/1", "/cards", "/inquiries/1", "/instant/j", "/start"]) {
      expect(backFallbackFor(path)).toBe("/");
    }
    expect(backFallbackFor("/chapters/new")).toBe("/map");
    expect(backFallbackFor("/settings")).toBe("/");
    expect(backFallbackFor("/settings/prompts")).toBe("/");
  });

  it("puts the Claude banner only on screens that generate", () => {
    for (const path of ["/", "/lectures/1", "/check/1", "/inquiries/1", "/chapters/new", "/instant/j", "/start"]) {
      expect(usesGeneration(path)).toBe(true);
    }
    for (const path of ["/map", "/history", "/cards", "/settings", "/settings/prompts"]) {
      expect(usesGeneration(path)).toBe(false);
    }
  });

  it("titles each path, 오늘 included", () => {
    expect(t[titleKeyFor("/")]).toBe("오늘");
    expect(t[titleKeyFor("/map")]).toBe("지도");
    expect(t[titleKeyFor("/history")]).toBe("기록");
    expect(t[titleKeyFor("/settings")]).toBe("punisher 설정");
    expect(t[titleKeyFor("/settings/prompts")]).toBe("강의 프롬프트");
    expect(t[titleKeyFor("/nope")]).toBe("찾을 수 없어요");
  });
});

describe("page actions", () => {
  it("adds punisher 설정 last, except on the settings screen", () => {
    const list = withAppSettings([{ id: "new", label: "새 챕터", kind: "default" }], "/map", t.appSettings);
    expect(list.map((a) => a.label)).toEqual(["새 챕터", "punisher 설정"]);
    expect(list[1]).toMatchObject({ kind: "settings", href: "/settings" });
    expect(withAppSettings(null, "/settings", t.appSettings)).toEqual([]);
    expect(withAppSettings(null, "/settings/prompts", t.appSettings).map((a) => a.label)).toEqual(["punisher 설정"]);
    // 셸 ⋯ 상시 "검색"은 없다
    expect(withAppSettings(null, "/", t.appSettings).some((a) => a.id === "search")).toBe(false);
  });

  const actions = withAppSettings([
    { id: "a", label: "A", kind: "default" },
    { id: "b", label: "B", kind: "default", icon: "plus" },
    { id: "c", label: "C", kind: "default", icon: "map" },
    { id: "m", label: "M" },
    { id: "x", label: "X", danger: true },
  ], "/map", t.appSettings);

  it("desktop: first two defaults as text buttons, then 설정, rest in ⋯ (danger stays inside — Menu puts it last)", () => {
    const { buttons, settings, menu } = layoutActions(actions, true);
    expect(buttons.map((a) => a.id)).toEqual(["a", "b"]);
    expect(settings?.id).toBe("settings");
    expect(menu.map((a) => a.id)).toEqual(["c", "m", "x"]);
  });

  it("phone: the first default with an icon, then ⋯ = rest → settings (Menu moves danger to the bottom)", () => {
    const { buttons, settings, menu } = layoutActions(actions, false);
    expect(buttons.map((a) => a.id)).toEqual(["b"]);
    expect(settings).toBeNull();
    expect(menu.map((a) => a.id)).toEqual(["a", "c", "m", "x", "settings"]);
  });

  it("never takes a danger action out of ⋯, even a default one", () => {
    const { buttons } = layoutActions([{ id: "d", label: "D", kind: "default", icon: "plus", danger: true }], false);
    expect(buttons).toEqual([]);
  });
});
