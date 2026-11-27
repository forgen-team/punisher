import type { Plugin as PostcssPlugin, Root } from "postcss";
import { isShellOwnedDeclaration, scopeSelector } from "./scope";

/**
 * Witems 빌드 전용 PostCSS 처리(vite.witems.config.ts). 화면 묶음에는 들어가지 않는다(postcss 는 타입만).
 *
 * 1. Shadow DOM 안에는 :root·html·body 가 없으므로 앱 루트(.punisher-root)로 바꾼다.
 * 2. 셸이 정하는 레이아웃 변수(--app-header-offset …)를 뺀다.
 * 3. 셸 토큰을 뺀다: Witems 는 design-tokens.css 의 변수(--color-*·--radius-*·--shadow-*·--font-* …)를 <html> 에 두고
 *    Shadow DOM 으로 이어 준다. 앱이 같은 이름을 .punisher-root 에 다시 선언하면 셸 값(인디고 강조색·다크 모드)을
 *    가리므로, design-tokens.css·brand.css 가 선언하는 이름은 Witems 빌드에서 모두 지운다. 단독판은 그대로 둔다.
 */
export function scopeToRoot(shellTokens: ReadonlySet<string>): PostcssPlugin {
  return {
    postcssPlugin: "punisher-scope-root",
    OnceExit(root: Root) {
      root.walkRules((rule) => {
        // keyframe steps (from/to/50%) are not selectors
        if (rule.parent?.type === "atrule" && /keyframes$/i.test((rule.parent as { name: string }).name)) { return; }
        rule.selectors = rule.selectors.map((selector) => scopeSelector(selector));
      });
      // 토큰 설명 주석(값 예시 포함)도 셸 안에는 필요 없다 — 빈 규칙 판정이 주석에 걸리지 않게 먼저 지운다.
      root.walkComments((comment) => { comment.remove(); });
      root.walkDecls((decl) => {
        if (isShellOwnedDeclaration(decl.prop) || shellTokens.has(decl.prop)) { decl.remove(); }
      });
      // 토큰만 있던 규칙(:root{…}, [data-theme=dark]{…})이 비면 지우고, 그 결과 빈 @media 도 지운다.
      root.walkRules((rule) => {
        if (rule.nodes.length === 0) { rule.remove(); }
      });
      root.walkAtRules((rule) => {
        if (rule.nodes && rule.nodes.length === 0) { rule.remove(); }
      });
    },
  };
}

/** CSS 글에서 선언된 사용자 정의 속성 이름(--x: …)을 모은다. 주석 안의 예시는 세지 않는다. */
export function declaredCustomProperties(...cssTexts: string[]): Set<string> {
  const names = new Set<string>();
  for (const css of cssTexts) {
    const code = css.replace(/\/\*[\s\S]*?\*\//g, "");
    for (const match of code.matchAll(/(--[\w-]+)\s*:/g)) { names.add(match[1]); }
  }
  return names;
}
