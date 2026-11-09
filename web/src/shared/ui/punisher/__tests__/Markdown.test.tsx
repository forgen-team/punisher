import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it } from "vitest";
import { Markdown } from "../Markdown";

function html(text: string): string {
  return renderToStaticMarkup(<Markdown renderTerm={(id, label) => <button data-term={id}>{label}</button>}>{text}</Markdown>);
}

describe("Markdown 굵게 — 한국어 조사가 바로 붙는 경우", () => {
  it.each([
    ["이것을 **리스(lease)**라고 합니다.", "<strong>리스(lease)</strong>라고"],
    ["값은 **10%**입니다", "<strong>10%</strong>입니다"],
    ['**"따옴표"**는', "<strong>&quot;따옴표&quot;</strong>는"],
    ["**[원자적](term:os.atomic)**으로", '<strong><button data-term="os.atomic">원자적</button></strong>으로'],
  ])("%s", (text, expected) => {
    const out = html(text);
    expect(out).toContain(expected);
    expect(out).not.toContain("**");
  });

  it("keeps normal bold, inline code and arithmetic as they were", () => {
    expect(html("**리스**라고")).toContain("<strong>리스</strong>라고");
    expect(html("`a**b**c` 그대로")).toContain("<code");
    expect(html("`a**b**c` 그대로")).toContain("a**b**c");
    expect(html("2 ** 3 ** 2 는 거듭제곱")).toContain("2 ** 3 ** 2");
  });
});
