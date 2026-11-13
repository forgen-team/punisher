/** 줄 단위 비교(LCS) — 바뀐 줄 번호(1부터)를 전·후 각각 돌려준다. 블록당 60줄 이하라 O(n·m) 로 충분하다. */
export interface LineDiff {
  removed: number[];
  added: number[];
}

export function diffLines(before: string, after: string): LineDiff {
  const a = before.split("\n");
  const b = after.split("\n");
  // lcs[row][column] = a[row..], b[column..] 의 최장 공통 부분열 길이
  const lcs: number[][] = Array.from({ length: a.length + 1 }, () => new Array<number>(b.length + 1).fill(0));
  for (let row = a.length - 1; row >= 0; row -= 1) {
    for (let column = b.length - 1; column >= 0; column -= 1) {
      lcs[row][column] = a[row] === b[column] ? lcs[row + 1][column + 1] + 1 : Math.max(lcs[row + 1][column], lcs[row][column + 1]);
    }
  }
  const removed: number[] = [];
  const added: number[] = [];
  let row = 0;
  let column = 0;
  while (row < a.length && column < b.length) {
    if (a[row] === b[column]) {
      row += 1;
      column += 1;
    } else if (lcs[row + 1][column] >= lcs[row][column + 1]) {
      removed.push(row + 1);
      row += 1;
    } else {
      added.push(column + 1);
      column += 1;
    }
  }
  for (; row < a.length; row += 1) { removed.push(row + 1); }
  for (; column < b.length; column += 1) { added.push(column + 1); }
  return { removed, added };
}
