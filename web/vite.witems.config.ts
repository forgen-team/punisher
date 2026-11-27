import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { federation } from "@module-federation/vite";
import react from "@vitejs/plugin-react";
import autoprefixer from "autoprefixer";
import tailwindcss from "tailwindcss";
import { defineConfig } from "vite";
import { declaredCustomProperties, scopeToRoot } from "./src/witems/cssScope";

/**
 * Witems app mode build (witems docs/APP-CONTRACT.md): a Module Federation
 * remote named "punisher" exposing ./mount, served from /apps-static/punisher/
 * and embedded into the Go binary (internal/witems/dist, `make witems-web`).
 *
 * The app renders inside the shell's Shadow DOM, where :root, html and body
 * never match. scopeToRoot (src/witems/cssScope.ts) rewrites them to the
 * element mount() marks with .punisher-root (and data-theme), drops the layout
 * variables the shell owns (--app-header-offset …) and drops every design
 * token (design-tokens.css) and brand accent (brand.css) declaration, so the
 * shell's tokens — Witems indigo accent, its light/dark values — reach the app.
 * The standalone build (vite.config.ts) keeps its own tokens and orange accent.
 */
const styles = (name: string) => readFileSync(fileURLToPath(new URL(`./src/styles/${name}`, import.meta.url)), "utf8");
const SHELL_TOKENS = declaredCustomProperties(styles("design-tokens.css"), styles("brand.css"));

const root = fileURLToPath(new URL(".", import.meta.url));
const REACT_RANGE = "^18.3.0-0"; // Next 14 app router shares its bundled React canary (contract §5)

export default defineConfig({
  base: "/apps-static/punisher/",
  plugins: [
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
  css: { postcss: { plugins: [tailwindcss(), autoprefixer(), scopeToRoot(SHELL_TOKENS)] } },
  resolve: { alias: { "@": root } },
  build: {
    target: "esnext",
    outDir: fileURLToPath(new URL("../internal/witems/dist", import.meta.url)),
    emptyOutDir: false, // keep dist/.gitkeep; `make witems-web` clears old files
    chunkSizeWarningLimit: 2000,
    rollupOptions: { input: fileURLToPath(new URL("./witems.html", import.meta.url)) },
  },
});
