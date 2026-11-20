/** 문단 글 안의 한 문장 — start·end 는 원문 글자 위치(end 미포함), 앞뒤 공백은 뺀다. */
export interface SentenceSpan {
  start: number;
  end: number;
  text: string;
}

/**
 * 문장 끝:
 * - 문장 부호(. ? ! …) 뒤에 닫는 괄호·따옴표가 붙을 수 있고, 그 뒤가 공백이거나 글 끝일 때(3.14·node.js 같은 붙은 점은 끝이 아니다)
 * - 전각 문장 부호(。？！) — 뒤 공백 없이도
 * - 마침표 없이 끝난 한국어 종결 어미(니다, 어요/아요/에요/예요/해요/세요/네요/까요, 죠) 뒤에 공백이 올 때
 * - 줄바꿈
 * 한국어 어미는 화면 문구 검사(noStrayKorean)에 걸리지 않게 유니코드 이스케이프로 쓴다.
 */
const SENTENCE_END = new RegExp(
  [
    // 전각 문장 부호(。？！)는 뒤에 공백이 없어도 끝이다(중국어·일본어는 띄어 쓰지 않는다).
    "[\\u3002\\uFF1F\\uFF01]+[)\\]\"'\\u201D\\u2019\\u300D\\u300F]*",
    "[.?!\\u2026]+[)\\]\"'\\u201D\\u2019\\u300D\\u300F]*(?=\\s|$)",
    "(?:\\uB2C8\\uB2E4|[\\uC5B4\\uC544\\uC5D0\\uC608\\uD574\\uC138\\uB124\\uAE4C]\\uC694|\\uC8E0)(?=\\s)",
    "\\n+",
  ].join("|"),
  "g",
);

function trimmedSpan(text: string, from: number, to: number): SentenceSpan | null {
  let start = from;
  let end = to;
  while (start < end && /\s/.test(text[start] ?? "")) { start += 1; }
  while (end > start && /\s/.test(text[end - 1] ?? "")) { end -= 1; }
  return start < end ? { start, end, text: text.slice(start, end) } : null;
}

/** 문단 글을 문장으로 나눈다(빈 문장은 버린다). */
export function splitSentences(text: string): SentenceSpan[] {
  const spans: SentenceSpan[] = [];
  let last = 0;
  for (const match of text.matchAll(SENTENCE_END)) {
    const end = (match.index ?? 0) + match[0].length;
    const span = trimmedSpan(text, last, end);
    if (span) { spans.push(span); }
    last = end;
  }
  const tail = trimmedSpan(text, last, text.length);
  if (tail) { spans.push(tail); }
  return spans;
}

/** 글자 위치(탭한 곳)가 속한 문장. 문장 사이 공백을 누르면 뒤 문장, 글 끝 너머면 마지막 문장. */
export function sentenceAt(text: string, offset: number): SentenceSpan | null {
  const spans = splitSentences(text);
  if (spans.length === 0) { return null; }
  return spans.find((s) => offset < s.end) ?? spans[spans.length - 1] ?? null;
}
