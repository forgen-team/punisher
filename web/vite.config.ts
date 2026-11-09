import { fileURLToPath } from "node:url";
import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

// The build lands in internal/webui/dist and is embedded into the Go binary.
// `npm run dev` proxies /v1 to a running `punisher --no-browser` on :4710
// (override with PUNISHER_DEV_API, e.g. http://127.0.0.1:4712).
const root = fileURLToPath(new URL(".", import.meta.url));

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: { "@": root },
  },
  build: {
    outDir: fileURLToPath(new URL("../internal/webui/dist", import.meta.url)),
    // Keep dist/.gitkeep (go:embed needs the directory); the Makefile clears old files.
    emptyOutDir: false,
    chunkSizeWarningLimit: 2000,
  },
  server: {
    port: 5173,
    proxy: { "/v1": process.env.PUNISHER_DEV_API ?? "http://127.0.0.1:4710" },
  },
});
