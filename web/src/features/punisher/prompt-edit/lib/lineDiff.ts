/** 줄 단위 diff 한 줄. same: 그대로 · add: 내 본문에만 · del: 기본값에만 */
export interface DiffLine {
  kind: "same" | "add" | "del";
  text: string;
}

/** 줄 단위 LCS diff(기본값 → 내 본문). 프롬프트 블록은 수십~수백 줄이라 O(n·m) 로 충분하다. */
export function lineDiff(before: string, after: string): DiffLine[] {
  const a = before.split("\n");
  const b = after.split("\n");
  const n = a.length;
  const m = b.length;
  // lcs[row][col] = a[row:] 와 b[col:] 의 최장 공통 부분열 길이
  const lcs: number[][] = Array.from({ length: n + 1 }, () => new Array<number>(m + 1).fill(0));
  for (let row = n - 1; row >= 0; row--) {
    for (let col = m - 1; col >= 0; col--) {
      lcs[row]![col] = a[row] === b[col] ? lcs[row + 1]![col + 1]! + 1 : Math.max(lcs[row + 1]![col]!, lcs[row]![col + 1]!);
    }
  }
  const out: DiffLine[] = [];
  let row = 0;
  let col = 0;
  while (row < n && col < m) {
    if (a[row] === b[col]) {
      out.push({ kind: "same", text: a[row]! });
      row++;
      col++;
    } else if (lcs[row + 1]![col]! >= lcs[row]![col + 1]!) {
      out.push({ kind: "del", text: a[row]! });
      row++;
    } else {
      out.push({ kind: "add", text: b[col]! });
      col++;
    }
  }
  while (row < n) { out.push({ kind: "del", text: a[row++]! }); }
  while (col < m) { out.push({ kind: "add", text: b[col++]! }); }
  return out;
}

/** 바뀐 줄 수(추가+삭제). */
export function changedLines(diff: DiffLine[]): number {
  return diff.filter((d) => d.kind !== "same").length;
}

/** 블록이 모르는 {{변수}} 이름들(서버 prompts.UnknownVars 와 같은 규칙). */
export function findUnknownVariables(body: string, declared: string[]): string[] {
  const known = new Set(declared);
  const out: string[] = [];
  for (const match of body.matchAll(/\{\{([^}]*)\}\}/g)) {
    const name = (match[1] ?? "").trim();
    if (name && !known.has(name) && !out.includes(name)) { out.push(name); }
  }
  return out;
}

/** 피드백 제안 문구를 본문 끝에 붙인 초안(이미 들어 있으면 그대로). */
export function withSuggestion(body: string, suggestion: string): string {
  const text = suggestion.trim();
  if (!text || body.includes(text)) { return body; }
  return `${body.replace(/\s+$/, "")}\n\n## 내 피드백 반영\n${text}`;
}
