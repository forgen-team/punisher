import { fileURLToPath } from "node:url";
import { defineConfig } from "vitest/config";

// 렌더 테스트(@testing-library/react)는 witems 와 같이 jsdom 에서 돈다.
export default defineConfig({
  resolve: { alias: { "@": fileURLToPath(new URL(".", import.meta.url)) } },
  test: {
    include: ["src/**/__tests__/**/*.test.{ts,tsx}"],
    environment: "jsdom",
    setupFiles: ["./src/vitest.setup.ts"],
    // 토큰 CSS 를 ?raw 로 읽는 테스트(witems/__tests__/cssScope)가 빈 문자열이 아니라 원문을 받게 한다.
    css: { include: [/src\/styles\/.+\.css/] },
  },
});
