import { freezeAppHeader } from "@/src/shared/lib/appHeaderCollapse";
import { queryApp } from "@/src/shared/lib/dom-root";
import { sentenceAt } from "./sentences";

interface TextIndex {
  text: string;
  nodes: Array<{ node: Text; start: number }>;
}

/** 요소 안 글 노드를 이어 붙인 글과, 각 노드가 시작하는 위치. */
function indexText(root: Node): TextIndex {
  const walker = document.createTreeWalker(root, NodeFilter.SHOW_TEXT);
  const nodes: TextIndex["nodes"] = [];
  let text = "";
  for (let node = walker.nextNode(); node; node = walker.nextNode()) {
    const value = (node as Text).data;
    nodes.push({ node: node as Text, start: text.length });
    text += value;
  }
  return { text, nodes };
}

function locate(index: TextIndex, offset: number, preferNext: boolean): { node: Text; offset: number } | null {
  for (let i = 0; i < index.nodes.length; i += 1) {
    const entry = index.nodes[i];
    const end = entry.start + entry.node.data.length;
    if (offset < end || (offset === end && !preferNext) || i === index.nodes.length - 1) {
      return { node: entry.node, offset: Math.min(Math.max(0, offset - entry.start), entry.node.data.length) };
    }
  }
  return null;
}

function rangeFor(index: TextIndex, start: number, end: number): Range | null {
  const from = locate(index, start, true);
  const to = locate(index, end, false);
  if (!from || !to) { return null; }
  const range = document.createRange();
  range.setStart(from.node, from.offset);
  range.setEnd(to.node, to.offset);
  return range;
}

/** 공백을 한 칸으로 줄인 글과 원래 위치 표(선택 글자의 줄바꿈·공백이 본문과 달라도 찾게). */
function collapseSpaces(text: string): { text: string; map: number[] } {
  let out = "";
  const map: number[] = [];
  for (let i = 0; i < text.length; i += 1) {
    const ch = text[i] ?? "";
    if (/\s/.test(ch)) {
      if (out.endsWith(" ")) { continue; }
      out += " ";
    } else {
      out += ch;
    }
    map.push(i);
  }
  return { text: out, map };
}

/** 요소 안에서 인용 구절의 첫 위치(Range). 없으면 null. */
export function findQuoteRange(root: Element, quote: string): Range | null {
  const needle = collapseSpaces(quote.trim()).text;
  if (!needle) { return null; }
  const index = indexText(root);
  const hay = collapseSpaces(index.text);
  const at = hay.text.indexOf(needle);
  if (at < 0) { return null; }
  const start = hay.map[at] ?? 0;
  const endChar = hay.map[at + needle.length - 1] ?? start;
  return rangeFor(index, start, endChar + 1);
}

/** 문장 탭을 받지 않는 곳 — 링크·용어·버튼·입력·코드 블록·그림. */
const IGNORE_TAP = "a, button, input, textarea, select, summary, label, pre, svg, [role='button'], [contenteditable], [data-no-sentence-tap]";
/** 문장을 나눌 문단 단위. 없으면 가장 가까운 진도 앵커 블록. */
const PARAGRAPH = "p, li, td, th, dt, dd, blockquote, figcaption, h2, h3, h4, h5, h6";

export function isIgnoredTapTarget(target: EventTarget | null): boolean {
  return target instanceof Element && target.closest(IGNORE_TAP) !== null;
}

function caretAt(x: number, y: number, root?: Node): { node: Node; offset: number } | null {
  if (typeof document.caretPositionFromPoint === "function") {
    // Witems 는 화면을 Shadow DOM 에 띄운다 — shadowRoots 를 넘겨야 그 안의 글자를 돌려준다(미지원 브라우저는 무시).
    const shadowRoots = root instanceof ShadowRoot ? [root] : [];
    const pos = document.caretPositionFromPoint(x, y, { shadowRoots } as { shadowRoots: ShadowRoot[] });
    return pos ? { node: pos.offsetNode, offset: pos.offset } : null;
  }
  if (typeof document.caretRangeFromPoint === "function") {
    const range = document.caretRangeFromPoint(x, y);
    return range ? { node: range.startContainer, offset: range.startOffset } : null;
  }
  return null;
}

export interface TappedSentence {
  text: string;
  /** 문단이 속한 진도 앵커(data-lecture-anchor) */
  anchor: string;
  range: Range;
}

/** 화면 좌표(탭)의 문장 — 글자 위가 아니거나 문단을 못 찾으면 null. */
export function sentenceAtPoint(container: HTMLElement, x: number, y: number): TappedSentence | null {
  const caret = caretAt(x, y, container.getRootNode());
  if (!caret || caret.node.nodeType !== Node.TEXT_NODE || !container.contains(caret.node)) { return null; }
  const host = caret.node.parentElement;
  const paragraph = host?.closest(PARAGRAPH) ?? host?.closest("[data-lecture-anchor]");
  if (!paragraph || !container.contains(paragraph) || isIgnoredTapTarget(host)) { return null; }
  const index = indexText(paragraph);
  const entry = index.nodes.find((n) => n.node === caret.node);
  if (!entry) { return null; }
  const span = sentenceAt(index.text, entry.start + caret.offset);
  if (!span) { return null; }
  const range = rangeFor(index, span.start, span.end);
  if (!range) { return null; }
  const anchor = paragraph.closest<HTMLElement>("[data-lecture-anchor]")?.dataset.lectureAnchor ?? "";
  return { text: span.text.replace(/\s+/g, " "), anchor, range };
}

/** CSS Custom Highlight API 를 쓸 수 있는가 — 없으면 고른 문장을 칠하지 못하므로 액션 바에 한 줄로 보여 준다. */
export function supportsHighlight(): boolean {
  return typeof CSS !== "undefined" && "highlights" in CSS && typeof Highlight !== "undefined";
}

/** CSS Custom Highlight API 로 본문을 바꾸지 않고 칠한다(지원하지 않는 브라우저는 칠하지 않는다). */
export function paintHighlight(name: string, ranges: Range[]): void {
  if (!supportsHighlight()) { return; }
  if (ranges.length === 0) {
    CSS.highlights.delete(name);
    return;
  }
  CSS.highlights.set(name, new Highlight(...ranges));
}

/**
 * 구절을 상단 바 바로 아래로 올려, 절반 높이 시트 위에 보이게 한다.
 * 휴대폰에선 스크롤 방향에 따라 셸 바가 숨거나 나타나는데, 그러면 강의 바 위치가 바뀌어 맞춘 자리가 어긋나므로
 * 이 스크롤 동안은 셸 바를 지금 상태로 둔다(지금 보이는 강의 바 아래에 그대로 놓인다).
 */
export function scrollRangeBelowTopBar(range: Range): void {
  const bar = queryApp("[data-lecture-readbar]", range.startContainer)?.getBoundingClientRect().bottom ?? 0;
  const delta = range.getBoundingClientRect().top - bar - 12;
  if (Math.abs(delta) > 4) {
    freezeAppHeader();
    window.scrollBy({ top: delta, behavior: "smooth" });
  }
}
