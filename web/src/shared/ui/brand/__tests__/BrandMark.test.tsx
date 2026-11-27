import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { BRAND_LINE, BrandMark } from "../BrandMark";

describe("BrandMark", () => {
  it("워드마크는 제품 이름 표기 그대로 소문자이고 CSS 로 대문자로 바꾸지 않는다", () => {
    render(<BrandMark />);
    const word = screen.getByText("punisher");
    expect(word.className).not.toMatch(/uppercase/);
    expect(screen.queryByText("PUNISHER")).toBeNull();
  });

  it("브랜드 한 줄도 소문자 punisher 로 시작한다", () => {
    expect(BRAND_LINE.startsWith("punisher ·")).toBe(true);
  });
});
