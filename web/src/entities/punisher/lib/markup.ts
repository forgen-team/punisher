// Client-side, lenient port of apps/punisher-server/internal/lecture/markup.go
// used only to render an instant lecture progressively while it streams.
// The saved lecture (validated server-side) replaces this view when done.
import type { BlockType, LectureBlock, LectureContent, LectureSection } from "@/src/shared/api/punisher";

const DIRECTIVE_RE = /^@@([a-z_]+)(?:\s+(.*))?$/;
const SOURCE_RE = /^\s*[-*]\s*\[([^\]]+)\]\((\S+?)\)\s*$/;
const BLOCK_TYPES = new Set<BlockType>(["markdown", "mermaid", "code", "log", "misconception", "term", "sources"]);

function parseAttributes(args: string): Record<string, string> {
  const attributes: Record<string, string> = {};
  let rest = args.trim();
  while (rest) {
    const greedy = ["title=", "term="].find((g) => rest.startsWith(g));
    if (greedy) {
      attributes[greedy.slice(0, -1)] = rest.slice(greedy.length).trim().replace(/^"|"$/g, "");
      return attributes;
    }
    const space = rest.search(/\s/);
    const token = space === -1 ? rest : rest.slice(0, space);
    rest = space === -1 ? "" : rest.slice(space).trim();
    const eq = token.indexOf("=");
    if (eq > 0) { attributes[token.slice(0, eq)] = token.slice(eq + 1).replace(/^"|"$/g, ""); }
    else if (!("_" in attributes)) { attributes._ = token; }
    else { attributes[token] = ""; }
  }
  return attributes;
}

function emptyContent(): LectureContent {
  return {
    schemaVersion: 1,
    title: "",
    conceptId: "",
    objectiveIds: [],
    depth: 1,
    estimatedMinutes: 0,
    newScope: [],
    prerequisitesReferenced: [],
    sections: [],
  };
}

function splitList(value: string): string[] {
  return value
    .split(",")
    .map((v) => v.trim())
    .filter(Boolean);
}

/** Parses possibly-incomplete markup into renderable content. */
export function parsePartialMarkup(text: string): LectureContent {
  const content = emptyContent();
  let section: LectureSection | null = null;
  let block: LectureBlock | null = null;
  let body: string[] = [];
  let inMeta = false;
  let blockCount = 0;

  const flush = () => {
    if (!block || !section) { return; }
    if (block.type === "sources") {
      block.sources = body.flatMap((line) => {
        const m = SOURCE_RE.exec(line);
        return m ? [{ title: m[1], url: m[2] }] : [];
      });
    } else {
      block.text = body.join("\n").replace(/^\n+|\n+$/g, "");
    }
    section.blocks.push(block);
    block = null;
    body = [];
  };

  for (const rawLine of text.replace(/\r\n/g, "\n").split("\n")) {
    const directive = DIRECTIVE_RE.exec(rawLine.trim());
    if (!directive) {
      if (inMeta) {
        const index = rawLine.indexOf(":");
        if (index > 0) {
          const key = rawLine.slice(0, index).trim();
          const value = rawLine.slice(index + 1).trim();
          switch (key) {
          case "title": {
          content.title = value;
          break;
          }
          case "objectives": {
          content.objectiveIds = splitList(value);
          break;
          }
          case "new_scope": {
          content.newScope.push(value);
          break;
          }
          case "prerequisites": {
          content.prerequisitesReferenced = splitList(value);
          // No default
          }
          break;
          }
        }
      } else if (block) {
        body.push(rawLine);
      }
      continue;
    }
    const [, word, args = ""] = directive;
    flush();
    if (word === "lecture") {
      inMeta = true;
      continue;
    }
    inMeta = false;
    if (word === "end") { break; }
    if (word === "section") {
      const attributes = parseAttributes(args);
      const kind = attributes._ ?? `section-${content.sections.length + 1}`;
      section = { anchor: kind, kind, title: attributes.title ?? "", minutes: Number(attributes.minutes) || 0, blocks: [] };
      content.sections.push(section);
      continue;
    }
    if (section && BLOCK_TYPES.has(word as BlockType)) {
      const attributes = parseAttributes(args);
      blockCount += 1;
      block = {
        anchor: `${section.anchor}-b${blockCount}`,
        type: word as BlockType,
        lang: attributes.lang,
        conceptId: attributes.concept,
        term: attributes.term,
        proposed: "proposed" in attributes,
      };
    }
  }
  flush();
  content.estimatedMinutes = content.sections.reduce((sum, s) => sum + s.minutes, 0);
  return content;
}
