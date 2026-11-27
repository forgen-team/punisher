import { fileURLToPath } from "node:url";
import { defineConfig } from "vitest/config";

// 렌더 테스트(@testing-library/react)는 witems 와 같이 jsdom 에서 돈다.
export default defineConfig({
  resolve: { alias: { "@": fileURLToPath(new URL(".", import.meta.url)) } },
  test: {
    include: ["src/**/__tests__/**/*.test.{ts,tsx}"],
    environment: "jsdom",
    setupFiles: ["./src/vitest.setup.ts"],
  },
});
