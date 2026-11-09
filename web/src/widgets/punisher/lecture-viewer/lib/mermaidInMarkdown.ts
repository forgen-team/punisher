/**
 * 마크다운 블록 안에 섞여 온 mermaid 도식을 떼어 낸다 — 모델이 @@mermaid 대신 ```mermaid 펜스나 펜스 없는
 * "sequenceDiagram" 줄로 쓰면 그림 대신 코드로 보이기 때문(2026-11-09). 서버 validate 의 MermaidOutsideBlock 과
 * 같은 규칙으로 찾고, 렌더러는 그 부분을 도식으로 그린다(이미 저장된 강의도 바로 보이게).
 */

export type MarkdownPart = { kind: "markdown" | "mermaid"; text: string };

/** 줄 전체가 도식 머리글일 때만 — 문장 안의 언급("sequenceDiagram 은 …")은 아니다. */
const HEADER = /^\s*(sequenceDiagram|classDiagram|erDiagram|gantt|journey|mindmap|timeline|stateDiagram(-v2)?|pie(\s+(showData|title\b.*))?|(flowchart|graph)\s+(TD|TB|BT|RL|LR))\s*$/;
const FENCE = /^(`{3,}|~{3,})(.*)$/;

function push(parts: MarkdownPart[], kind: MarkdownPart["kind"], lines: string[]) {
  const text = lines.join("\n");
  if (!text.trim()) { return; }
  const prev = parts[parts.length - 1];
  if (prev && prev.kind === kind && kind === "markdown") { prev.text += `\n${text}`; }
  else { parts.push({ kind, text: kind === "mermaid" ? text.trim() : text }); }
}

export function splitMermaid(text: string): MarkdownPart[] {
  if (!/sequenceDiagram|classDiagram|erDiagram|gantt|journey|mindmap|timeline|stateDiagram|pie|flowchart|graph|mermaid/.test(text)) {
    return [{ kind: "markdown", text }];
  }
  const lines = text.split("\n");
  const parts: MarkdownPart[] = [];
  let md: string[] = [];
  for (let at = 0; at < lines.length; at++) {
    const line = lines[at];
    const trimmed = line.replace(/^ {0,3}/, "");
    const fence = line.length - trimmed.length <= 3 ? FENCE.exec(trimmed) : null;
    if (fence) {
      // 펜스 끝 찾기(같은 문자, 같거나 긴 길이). 없으면 끝까지.
      const marker = fence[1];
      let end = lines.length;
      for (let next = at + 1; next < lines.length; next++) {
        const t = lines[next].replace(/^ {0,3}/, "");
        const m = FENCE.exec(t);
        if (m && m[1][0] === marker[0] && m[1].length >= marker.length && m[2].trim() === "") { end = next; break; }
      }
      const lang = fence[2].trim().toLowerCase();
      const body = lines.slice(at + 1, end);
      const firstLine = body.find((l) => l.trim() !== "") ?? "";
      if (lang === "mermaid" || ((lang === "" || lang === "text") && HEADER.test(firstLine))) {
        push(parts, "markdown", md);
        md = [];
        push(parts, "mermaid", body);
      } else {
        md.push(...lines.slice(at, Math.min(end + 1, lines.length)));
      }
      at = end;
      continue;
    }
    if (HEADER.test(line) && line.length - trimmed.length <= 3) {
      // 펜스 없는 도식: 머리글부터 빈 줄 전까지
      let end = at + 1;
      while (end < lines.length && lines[end].trim() !== "") { end++; }
      push(parts, "markdown", md);
      md = [];
      push(parts, "mermaid", lines.slice(at, end));
      at = end - 1;
      continue;
    }
    md.push(line);
  }
  push(parts, "markdown", md);
  return parts.length > 0 ? parts : [{ kind: "markdown", text }];
}
