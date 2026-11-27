// witems apps/web/src/vitest.setup.ts 와 같은 설정: jest-dom 매처, 테스트마다 렌더 정리.
import "@testing-library/jest-dom/vitest";
import { createElement, type ComponentType, type ReactNode } from "react";
import { afterEach, vi } from "vitest";
import { cleanup } from "@testing-library/react";

// 화면 부품(@forgen-team/ui)은 UIProvider·FeedbackProvider 아래에서 돈다(useConfirm·useToast 는 공급자가 없으면 throw).
// 앱 뿌리(app/UIRoot.tsx)와 같게, 모든 render·renderHook 을 두 공급자로 감싼다. 테스트가 준 wrapper 는 그 안에 둔다.
vi.mock("@testing-library/react", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@testing-library/react")>();
  const { UIProvider, FeedbackProvider } = await import("@forgen-team/ui");
  const withUI = (inner?: ComponentType<{ children: ReactNode }>) =>
    function TestUI({ children }: { children: ReactNode }) {
      const content = inner ? createElement(inner, null, children) : children;
      return createElement(UIProvider, null, createElement(FeedbackProvider, null, content));
    };
  return {
    ...actual,
    render: ((ui: Parameters<typeof actual.render>[0], options?: Parameters<typeof actual.render>[1]) => actual.render(ui, { ...options, wrapper: withUI(options?.wrapper as ComponentType<{ children: ReactNode }> | undefined) })) as typeof actual.render,
    renderHook: ((hook, options) => actual.renderHook(hook, { ...options, wrapper: withUI(options?.wrapper as ComponentType<{ children: ReactNode }> | undefined) })) as typeof actual.renderHook,
  };
});

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

// @forgen-team/ui(SEED) 가 쓰는데 jsdom 에 없는 것들 — 디자인 패키지 test/setup.ts 와 같다
class ResizeObserverStub { observe() {} unobserve() {} disconnect() {} }
(globalThis as unknown as { ResizeObserver: typeof ResizeObserverStub }).ResizeObserver ??= ResizeObserverStub;
(Element.prototype as unknown as { scrollIntoView: () => void }).scrollIntoView ??= () => {};
(HTMLElement.prototype as unknown as { hasPointerCapture: () => boolean }).hasPointerCapture ??= () => false;
const cssGlobal = globalThis as unknown as { CSS?: { supports?: (...a: string[]) => boolean } };
cssGlobal.CSS ??= {};
cssGlobal.CSS.supports ??= () => false;
