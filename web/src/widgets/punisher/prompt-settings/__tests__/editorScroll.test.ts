import { describe, expect, it } from "vitest";
import { editorOutOfView } from "../lib/editorScroll";

describe("editorOutOfView", () => {
  it("편집기 머리가 화면 위로 지나갔거나 아래쪽에 있으면 스크롤한다", () => {
    expect(editorOutOfView(-400, 800)).toBe(true);
    expect(editorOutOfView(700, 800)).toBe(true);
  });

  it("화면 위쪽에 보이면 그대로 둔다(상단 바 높이 고려)", () => {
    expect(editorOutOfView(120, 800)).toBe(false);
    expect(editorOutOfView(30, 800, 48)).toBe(true);
  });
});
