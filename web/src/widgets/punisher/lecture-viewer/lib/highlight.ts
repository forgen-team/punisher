// Loaded only via dynamic import() from CodeBlock so shiki stays out of the
// route's initial bundle. Uses the JS regex engine (no WASM fetch) and a
// curated language set (the full shiki bundle would add ~13 MB of grammars
// to the embedded binary); unknown languages render as plain text.
import { createHighlighterCore, type HighlighterCore, type LanguageInput } from "shiki/core";
import { createJavaScriptRegexEngine } from "shiki/engine/javascript";

const LANGUAGES: Record<string, LanguageInput> = {
  bash: () => import("shiki/langs/bash.mjs"),
  c: () => import("shiki/langs/c.mjs"),
  cpp: () => import("shiki/langs/cpp.mjs"),
  csharp: () => import("shiki/langs/csharp.mjs"),
  css: () => import("shiki/langs/css.mjs"),
  diff: () => import("shiki/langs/diff.mjs"),
  docker: () => import("shiki/langs/docker.mjs"),
  go: () => import("shiki/langs/go.mjs"),
  graphql: () => import("shiki/langs/graphql.mjs"),
  html: () => import("shiki/langs/html.mjs"),
  http: () => import("shiki/langs/http.mjs"),
  ini: () => import("shiki/langs/ini.mjs"),
  java: () => import("shiki/langs/java.mjs"),
  javascript: () => import("shiki/langs/javascript.mjs"),
  json: () => import("shiki/langs/json.mjs"),
  jsx: () => import("shiki/langs/jsx.mjs"),
  kotlin: () => import("shiki/langs/kotlin.mjs"),
  make: () => import("shiki/langs/make.mjs"),
  markdown: () => import("shiki/langs/markdown.mjs"),
  nginx: () => import("shiki/langs/nginx.mjs"),
  proto: () => import("shiki/langs/proto.mjs"),
  python: () => import("shiki/langs/python.mjs"),
  rust: () => import("shiki/langs/rust.mjs"),
  sql: () => import("shiki/langs/sql.mjs"),
  toml: () => import("shiki/langs/toml.mjs"),
  tsx: () => import("shiki/langs/tsx.mjs"),
  typescript: () => import("shiki/langs/typescript.mjs"),
  xml: () => import("shiki/langs/xml.mjs"),
  yaml: () => import("shiki/langs/yaml.mjs"),
};

const LANG_ALIASES: Record<string, string> = {
  golang: "go", shell: "bash", sh: "bash", zsh: "bash", console: "bash", js: "javascript", ts: "typescript",
  py: "python", kt: "kotlin", yml: "yaml", dockerfile: "docker", makefile: "make", "c++": "cpp", cs: "csharp",
  md: "markdown", protobuf: "proto", rs: "rust",
};

let highlighterPromise: Promise<HighlighterCore> | null = null;

function getHighlighter(): Promise<HighlighterCore> {
  highlighterPromise ??= createHighlighterCore({
    themes: [import("shiki/themes/github-light.mjs"), import("shiki/themes/github-dark.mjs")],
    langs: [],
    engine: createJavaScriptRegexEngine(),
  });
  return highlighterPromise;
}

export async function highlightToHtml(code: string, lang: string | undefined): Promise<string> {
  const highlighter = await getHighlighter();
  const requested = (lang ?? "").toLowerCase();
  const normalized = LANG_ALIASES[requested] ?? requested;
  let resolved = "text";
  const loader = LANGUAGES[normalized];
  if (loader) {
    resolved = normalized;
    if (!highlighter.getLoadedLanguages().includes(resolved)) {
      await highlighter.loadLanguage(loader);
    }
  }
  return highlighter.codeToHtml(code, {
    lang: resolved,
    themes: { light: "github-light", dark: "github-dark" },
    defaultColor: false,
  });
}
