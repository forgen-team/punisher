import { readFileSync } from "node:fs";
import { createRequire } from "node:module";
import { fileURLToPath } from "node:url";
import { federation } from "@module-federation/vite";
import shadowScope, { declaredCustomProperties } from "@forgen-team/postcss-shadow-scope";
import react from "@vitejs/plugin-react";
import autoprefixer from "autoprefixer";
import tailwindcss from "tailwindcss";
import { defineConfig } from "vite";
import { ROOT_SELECTOR } from "./src/witems/scope";
import { dropSeedRecipeCss } from "@forgen-team/ui/vite";

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
