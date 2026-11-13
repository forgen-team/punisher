const FENCE = /^\s{0,3}(`{3,}|~{3,})/;

/**
 * 단일 줄바꿈을 Markdown 강제 줄바꿈(줄 끝 공백 2개)으로 바꾼다 — 줄마다 나뉜 ①②③ 이 한 문단으로 붙지 않게.
 * 코드 펜스 안, 표 줄(`|` 로 시작), 빈 줄 앞뒤는 그대로 둔다.
 */
export function keepLineBreaks(text: string): string {
  const lines = text.split("\n");
  let fence = "";
  return lines
    .map((line, index) => {
      const marker = FENCE.exec(line)?.[1];
      if (marker && (!fence || (marker[0] === fence[0] && marker.length >= fence.length))) {
        fence = fence ? "" : marker;
        return line;
      }
      const next = lines[index + 1];
      if (fence || next === undefined || line.trim() === "" || next.trim() === "" || FENCE.test(next)) { return line; }
      if (line.trimStart().startsWith("|") || next.trimStart().startsWith("|")) { return line; }
      return `${line.trimEnd()}  `;
    })
    .join("\n");
}
