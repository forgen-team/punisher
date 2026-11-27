import { fileURLToPath } from "node:url";
import { defineConfig } from "vitest/config";

// 렌더 테스트(@testing-library/react)는 witems 와 같이 jsdom 에서 돈다.
export default defineConfig({
  resolve: { alias: { "@": fileURLToPath(new URL(".", import.meta.url)) } },
  test: {
    // @forgen-team/ui 가 쓰는 SEED recipe 모듈이 JS 안에서 CSS 를 import 한다 → Node 가 아니라 Vite 가 처리하게
    server: { deps: { inline: [/@seed-design\//, /@forgen-team\/ui/] } },
    include: ["src/**/__tests__/**/*.test.{ts,tsx}"],
    environment: "jsdom",
    setupFiles: ["./src/vitest.setup.ts"],
  },
});
