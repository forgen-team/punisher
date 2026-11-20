import { describe, expect, it } from "vitest";
import { sentenceAt, splitSentences } from "../lib/sentences";

const texts = (input: string) => splitSentences(input).map((s) => s.text);

describe("splitSentences", () => {
  it("splits on . ? ! before a space or the end, and on 。 even without a space", () => {
    expect(texts("TCP is reliable. Is UDP? Yes! 終わり。次")).toEqual(["TCP is reliable.", "Is UDP?", "Yes!", "終わり。", "次"]);
  });

  it("keeps decimals, versions and dotted names inside one sentence", () => {
    expect(texts("값은 3.14 입니다. node.js v1.2 를 씁니다.")).toEqual(["값은 3.14 입니다.", "node.js v1.2 를 씁니다."]);
  });

  it("splits after Korean sentence endings followed by a space even without punctuation", () => {
    expect(texts("캐시는 빠릅니다 하지만 작아요 그래서 교체가 필요하죠 끝")).toEqual(["캐시는 빠릅니다", "하지만 작아요", "그래서 교체가 필요하죠", "끝"]);
  });

  it("attaches closing quotes/brackets to the sentence and splits on newlines", () => {
    expect(texts('그는 "됩니다." 라고 했다.\n다음 줄(괄호.) 끝')).toEqual(['그는 "됩니다."', "라고 했다.", "다음 줄(괄호.)", "끝"]);
  });

  it("reports offsets into the original text without surrounding spaces", () => {
    const input = "  첫 문장. 둘째 문장.  ";
    const [first, second] = splitSentences(input);
    expect(first).toEqual({ start: 2, end: 7, text: "첫 문장." });
    expect(input.slice(second.start, second.end)).toBe("둘째 문장.");
  });

  it("returns nothing for blank text", () => {
    expect(splitSentences("   \n ")).toEqual([]);
  });
});

describe("sentenceAt", () => {
  const input = "첫 문장. 둘째 문장. 셋째";
  it("finds the sentence under the tapped offset", () => {
    expect(sentenceAt(input, 0)?.text).toBe("첫 문장.");
    expect(sentenceAt(input, input.indexOf("둘째") + 1)?.text).toBe("둘째 문장.");
  });
  it("picks the following sentence when tapping the gap and the last one past the end", () => {
    expect(sentenceAt(input, input.indexOf(" 둘째"))?.text).toBe("둘째 문장.");
    expect(sentenceAt(input, 999)?.text).toBe("셋째");
  });
  it("returns null for empty text", () => {
    expect(sentenceAt("", 0)).toBeNull();
  });
});
