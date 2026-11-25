/** `@@recall` 본문 — 질문 줄과 "떠올릴 것:"(영어 "Recall:") 줄(서버 lecture.RecallParts 와 같음). */
export interface RecallParts {
  question: string;
  key: string;
}

const KEY_PREFIXES = ["떠올릴 것:", "Recall:"];

export function parseRecall(text: string): RecallParts {
  const lines = text.split("\n");
  for (let i = 0; i < lines.length; i += 1) {
    const line = lines[i].trim();
    const prefix = KEY_PREFIXES.find((p) => line.startsWith(p));
    if (prefix) {
      const key = [line.slice(prefix.length).trim(), ...lines.slice(i + 1)].join("\n").trim();
      return { question: lines.slice(0, i).join("\n").trim(), key };
    }
  }
  return { question: text.trim(), key: "" };
}
