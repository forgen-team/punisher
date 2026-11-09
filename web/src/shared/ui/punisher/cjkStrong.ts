/**
 * CommonMark 의 강조 규칙은 닫는 ** 앞이 문장 부호이고 뒤가 글자(공백·문장 부호가 아님)면 닫지 못한다.
 * 한국어는 조사가 바로 붙어서 "**리스(lease)**라고", "**[원자적](term:x)**으로", "**10%**입니다" 가
 * 굵게가 되지 않고 별표가 그대로 보인다(2026-11-09 강의에서 확인). 파싱이 끝난 뒤 같은 부모 안에 남은
 * ** 쌍을 굵게(strong)로 바꾼다. 코드(inlineCode·code)는 text 노드가 아니라 건드리지 않는다.
 */

interface MdNode {
  type: string;
  value?: string;
  children?: MdNode[];
}

/** 문장 안 내용만 담는 부모 — 여기서만 ** 쌍을 맞춘다(블록 경계를 넘지 않음). */
const PHRASING_PARENTS = new Set(["paragraph", "heading", "tableCell", "emphasis", "link", "delete"]);
const MARKER = "**";

type Token = { marker: true } | { marker: false; node: MdNode };

function tokenize(children: MdNode[]): Token[] {
  const tokens: Token[] = [];
  for (const child of children) {
    if (child.type !== "text" || !child.value?.includes(MARKER)) {
      tokens.push({ marker: false, node: child });
      continue;
    }
    child.value.split(MARKER).forEach((part, index) => {
      if (index > 0) { tokens.push({ marker: true }); }
      if (part) { tokens.push({ marker: false, node: { type: "text", value: part } }); }
    });
  }
  return tokens;
}

function edgeIsSpace(nodes: MdNode[]): boolean {
  const first = nodes[0];
  const last = nodes[nodes.length - 1];
  return (first.type === "text" && /^\s/.test(first.value ?? "")) || (last.type === "text" && /\s$/.test(last.value ?? ""));
}

function mergeText(nodes: MdNode[]): MdNode[] {
  const out: MdNode[] = [];
  for (const node of nodes) {
    const prev = out[out.length - 1];
    if (node.type === "text" && prev?.type === "text") { prev.value = (prev.value ?? "") + (node.value ?? ""); }
    else { out.push(node); }
  }
  return out;
}

export function pairStrong(children: MdNode[]): MdNode[] {
  const tokens = tokenize(children);
  if (tokens.filter((t) => t.marker).length < 2) { return children; }
  const out: MdNode[] = [];
  for (let at = 0; at < tokens.length; at++) {
    const token = tokens[at];
    if (!token.marker) { out.push(token.node); continue; }
    const close = tokens.findIndex((t, next) => next > at && t.marker);
    const inner = close > at + 1 ? tokens.slice(at + 1, close).map((t) => (t.marker ? null : t.node)).filter((n): n is MdNode => n !== null) : [];
    // 빈 내용·앞뒤 공백("a ** b ** c" 같은 곱셈 표기)은 굵게가 아니다.
    if (close === -1 || inner.length === 0 || edgeIsSpace(inner)) {
      out.push({ type: "text", value: MARKER });
      continue;
    }
    out.push({ type: "strong", children: mergeText(inner) });
    at = close;
  }
  return mergeText(out);
}

function walk(node: MdNode): void {
  if (!node.children) { return; }
  for (const child of node.children) { walk(child); }
  if (PHRASING_PARENTS.has(node.type)) { node.children = pairStrong(node.children); }
}

/** remark 플러그인: CommonMark 가 닫지 못한 ** 쌍을 굵게로(한국어 조사가 붙는 경우). */
export function remarkCjkStrong() {
  return (tree: MdNode) => walk(tree);
}
