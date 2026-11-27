import { describe, expect, it } from "vitest";
import { BRAND_LINE } from "../BrandMark";

// 워드마크는 단독판 틀(AppFrame brand.name = "punisher")이 그린다 — 앱에는 로고(PunisherLogo)와 브랜드 한 줄만 남는다.
describe("brand", () => {
  it("브랜드 한 줄도 소문자 punisher 로 시작한다", () => {
    expect(BRAND_LINE.startsWith("punisher ·")).toBe(true);
  });
});
