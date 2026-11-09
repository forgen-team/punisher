/**
 * 닫히지 않은 ``` / ~~~ 코드 펜스를 본문 끝에서 닫는다 — 모델이 펜스를 닫지 않으면 이후 본문 전체가
 * 코드블록으로 렌더링되기 때문(서버 validate 의 UnclosedFence 와 같은 규칙).
 */
export function closeUnclosedFence(text: string): string {
  let open = "";
  for (const line of text.split("\n")) {
    const trimmed = line.replace(/^ {0,3}/, "");
    if (line.length - trimmed.length > 3) { continue; }
    const marker = /^(`{3,}|~{3,})/.exec(trimmed)?.[1];
    if (!marker) { continue; }
    if (!open) { open = marker; }
    else if (marker[0] === open[0] && marker.length >= open.length && trimmed.slice(marker.length).trim() === "") { open = ""; }
  }
  return open ? `${text.replace(/\s+$/, "")}\n${open}` : text;
}
