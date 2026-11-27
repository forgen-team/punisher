
import { useEffect, useRef, type RefObject } from "react";
import { activeAppElement } from "@/src/shared/lib/dom-root";

const FOCUSABLE =
  'a[href], button:not([disabled]), input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])';

/**
 * 열린 동안 Tab 포커스를 container 안에 가두고, Esc로 onEscape를 호출한다.
 * 닫히면 열기 직전 포커스 요소로 되돌린다.
 */
export function useFocusTrap(
  containerRef: RefObject<HTMLElement>,
  active: boolean,
  onEscape?: () => void
) {
  // onEscape는 ref로 보관: 부모 리렌더(새 콜백)마다 포커스가 처음으로 튀지 않게 한다
  const escapeRef = useRef(onEscape);
  useEffect(() => {
    escapeRef.current = onEscape;
  }, [onEscape]);

  useEffect(() => {
    if (!active) return;
    const container = containerRef.current;
    if (!container) return;
    // Shadow DOM 안에서는 document.activeElement 가 호스트다 — 컨테이너가 속한 루트의 activeElement 를 쓴다.
    const focused = () => activeAppElement(container);
    const previouslyFocused = focused();

    const focusables = () =>
      Array.from(container.querySelectorAll<HTMLElement>(FOCUSABLE)).filter(
        (el) => el.offsetParent !== null || el === focused()
      );

    const initial = container.querySelector<HTMLElement>("[data-autofocus]") ?? focusables()[0] ?? container;
    initial.focus({ preventScroll: true });

    function handleKeyDown(event: KeyboardEvent) {
      if (event.key === "Escape") {
        event.stopPropagation();
        escapeRef.current?.();
        return;
      }
      if (event.key !== "Tab" || !container) return;
      const items = focusables();
      if (items.length === 0) {
        event.preventDefault();
        return;
      }
      const first = items[0]!;
      const last = items[items.length - 1]!;
      if (event.shiftKey && focused() === first) {
        event.preventDefault();
        last.focus();
      } else if (!event.shiftKey && focused() === last) {
        event.preventDefault();
        first.focus();
      }
    }

    container.addEventListener("keydown", handleKeyDown);
    return () => {
      container.removeEventListener("keydown", handleKeyDown);
      previouslyFocused?.focus?.({ preventScroll: true });
    };
  }, [active, containerRef, escapeRef]);
}
