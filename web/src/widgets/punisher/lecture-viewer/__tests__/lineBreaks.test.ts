import { describe, expect, it } from "vitest";
import { keepLineBreaks } from "../lib/lineBreaks";

describe("keepLineBreaks", () => {
  it("단일 줄바꿈을 강제 줄바꿈으로 바꾼다", () => {
    expect(keepLineBreaks("① 하나\n② 둘\n③ 셋")).toBe("① 하나  \n② 둘  \n③ 셋");
  });

  it("빈 줄(문단 구분)과 코드 펜스 안은 그대로 둔다", () => {
    expect(keepLineBreaks("가\n\n나")).toBe("가\n\n나");
    expect(keepLineBreaks("설명\n```\na\nb\n```\n끝")).toBe("설명\n```\na\nb\n```\n끝");
  });

  it("표 줄은 그대로 둔다", () => {
    const table = "| a | b |\n| - | - |\n| 1 | 2 |";
    expect(keepLineBreaks(table)).toBe(table);
  });
});
