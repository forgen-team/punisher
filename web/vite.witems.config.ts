import { readFileSync } from "node:fs";
import { createRequire } from "node:module";
import { fileURLToPath } from "node:url";
import { federation } from "@module-federation/vite";
import shadowScope, { declaredCustomProperties } from "@forgen-team/postcss-shadow-scope";
import react from "@vitejs/plugin-react";
import autoprefixer from "autoprefixer";
import tailwindcss from "tailwindcss";
import { defineConfig, type Plugin } from "vite";
import { ROOT_SELECTOR } from "./src/witems/scope";

/**
 * Witems app mode build (witems docs/APP-CONTRACT.md): a Module Federation
 * remote named "punisher" exposing ./mount, served from /apps-static/punisher/
 * and embedded into the Go binary (internal/witems/dist, `make witems-web`).
 *
 * The app renders inside the shell's Shadow DOM, where :root, html and body
 * never match. @forgen-team/postcss-shadow-scope rewrites them to the element
 * mount() marks with .punisher-root (and data-theme), drops the layout
 * variables the shell owns (--app-header-offset …), every design token
 * declared by @forgen-team/tokens (tokens.css, seed.css) and the SEED root
 * variables (--seed-*), so the shell's tokens — its accent, light/dark values —
 * reach the app. The standalone build (vite.config.ts) keeps the tokens.
 */
const require = createRequire(import.meta.url);
const SHELL_TOKENS = declaredCustomProperties(
  readFileSync(require.resolve("@forgen-team/tokens/tokens.css"), "utf8"),
  readFileSync(require.resolve("@forgen-team/tokens/seed.css"), "utf8"),
);

/**
 * SEED recipe 모듈(@seed-design/css/recipes/*.mjs, @forgen-team/ui 가 씀)은 자기 CSS 를 `import "./x.css"` 로 부른다.
 * 그대로 두면 Vite 가 따로 CSS 파일로 빼서 셸 <head>(Shadow DOM 밖)에 넣는다 — make witems-web 의 stray CSS 검사.
 * 같은 CSS 는 globals.css 의 @forgen-team/ui/all.css(@seed-design/css/all.css)로 Shadow DOM 안에 이미 있으므로 비운다.
 * (chiron·atlas 와 같은 코드 — 패키지로 옮길 예정, design PLANS.md 후속)
 */
const EMPTY_SEED_CSS = "\0punisher-empty-seed-recipe-css";
function dropSeedRecipeCss(): Plugin {
  return {
    name: "punisher-drop-seed-recipe-css",
    enforce: "pre",
    resolveId(source, importer) {
      if (importer && source.endsWith(".css") && /[\\/]@seed-design[\\/]css[\\/]/.test(importer)) { return EMPTY_SEED_CSS; }
      return null;
    },
    load(id) { return id === EMPTY_SEED_CSS ? "" : null; },
  };
}

const root = fileURLToPath(new URL(".", import.meta.url));
const REACT_RANGE = "^18.3.0-0"; // Next 14 app router shares its bundled React canary (contract §5)

export default defineConfig({
  base: "/apps-static/punisher/",
  plugins: [
    dropSeedRecipeCss(),
    react(),
    federation({
      name: "punisher",
      filename: "remoteEntry.js",
      exposes: { "./mount": "./src/witems/mount.tsx" },
      shared: {
        react: { singleton: true, requiredVersion: REACT_RANGE },
        "react-dom": { singleton: true, requiredVersion: REACT_RANGE },
      },
      dts: false,
    }),
  ],
  css: {
    postcss: {
      plugins: [tailwindcss(), autoprefixer(), shadowScope({ root: ROOT_SELECTOR, tokens: SHELL_TOKENS, stripPrefixes: ["--seed-"] })],
    },
  },
  resolve: { alias: { "@": root } },
  build: {
    target: "esnext",
    outDir: fileURLToPath(new URL("../internal/witems/dist", import.meta.url)),
    emptyOutDir: false, // keep dist/.gitkeep; `make witems-web` clears old files
    chunkSizeWarningLimit: 2000,
    rollupOptions: { input: fileURLToPath(new URL("./witems.html", import.meta.url)) },
  },
});
