import type { LabOS } from "@/src/shared/api/punisher";

/** A lab body: the "$ command" line, then 보는 곳: / 예상: lines (server lecture.LabCommand). */
export interface LabParts {
  command: string;
  look: string;
  expect: string;
}

export function parseLab(text: string): LabParts {
  const parts: LabParts = { command: "", look: "", expect: "" };
  for (const raw of text.split("\n")) {
    const line = raw.trim();
    if (line.startsWith("$ ") && !parts.command) { parts.command = line.slice(2).trim(); }
    else if (line.startsWith("보는 곳:")) { parts.look = line.slice("보는 곳:".length).trim(); }
    else if (line.startsWith("예상:")) { parts.expect = line.slice("예상:".length).trim(); }
  }
  return parts;
}

/** Whether a lab block (os = macos | linux | any) is for the reader's environment. */
export function labForOS(blockOS: string | undefined, readerOS: LabOS | undefined): boolean {
  if (!readerOS || readerOS === "none") { return blockOS === "any" || blockOS === "macos"; }
  return blockOS === "any" || blockOS === readerOS;
}
