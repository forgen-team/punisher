import { fileURLToPath } from "node:url";
import { federation } from "@module-federation/vite";
import react from "@vitejs/plugin-react";
import autoprefixer from "autoprefixer";
import type { Plugin as PostcssPlugin, Root } from "postcss";
import tailwindcss from "tailwindcss";
import { defineConfig } from "vite";
import { scopeSelector } from "./src/witems/scope";

/**
 * Witems app mode build (witems docs/APP-CONTRACT.md): a Module Federation
 * remote named "punisher" exposing ./mount, served from /apps-static/punisher/
 * and embedded into the Go binary (internal/witems/dist, `make witems-web`).
 *
 * The app renders inside the shell's Shadow DOM, where :root, html and body
 * never match. scopeToRoot rewrites them to the element mount() marks with
 * .punisher-root (and data-theme), so tokens, dark mode and base styles apply.
 */
function scopeToRoot(): PostcssPlugin {
  return {
    postcssPlugin: "punisher-scope-root",
    OnceExit(root: Root) {
      root.walkRules((rule) => {
        // keyframe steps (from/to/50%) are not selectors
        if (rule.parent?.type === "atrule" && /keyframes$/i.test((rule.parent as { name: string }).name)) { return; }
        rule.selectors = rule.selectors.map(scopeSelector);
      });
    },
  };
}

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
  css: { postcss: { plugins: [tailwindcss(), autoprefixer(), scopeToRoot()] } },
  resolve: { alias: { "@": root } },
  build: {
    target: "esnext",
    outDir: fileURLToPath(new URL("../internal/witems/dist", import.meta.url)),
    emptyOutDir: false, // keep dist/.gitkeep; `make witems-web` clears old files
    chunkSizeWarningLimit: 2000,
    rollupOptions: { input: fileURLToPath(new URL("./witems.html", import.meta.url)) },
  },
});
