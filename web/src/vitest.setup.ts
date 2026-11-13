// witems apps/web/src/vitest.setup.ts 와 같은 설정: jest-dom 매처, 테스트마다 렌더 정리.
import "@testing-library/jest-dom/vitest";
import { afterEach, vi } from "vitest";
import { cleanup } from "@testing-library/react";

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});
