import { witemsRuntime } from "@/src/shared/config/runtime";

/**
 * 화면이 그려지는 루트 노드 — 단독판은 document, Witems 는 셸이 만든 Shadow DOM(계약 §3·§4).
 * Shadow DOM 안의 요소는 document.getElementById·querySelector 로 찾을 수 없고, document 에 단
 * 이벤트의 target 은 호스트 요소로 바뀌며(retarget), document.activeElement 도 호스트를 가리킨다.
 * 이 파일의 함수로 조회하면 두 모드가 같게 동작한다.
 */
export type AppRoot = Document | ShadowRoot;

/** 지금 화면의 루트. node 를 주면 그 노드의 루트(문서에서 떨어진 노드면 document). */
export function appRoot(node?: Node | null): AppRoot {
  const base = node ?? witemsRuntime()?.portal ?? null;
  const root = base?.getRootNode();
  if (root instanceof ShadowRoot || root instanceof Document) { return root; }
  return document;
}

export function getAppElementById(id: string, node?: Node | null): HTMLElement | null {
  const root = appRoot(node);
  const found = root.getElementById(id);
  return found instanceof HTMLElement ? found : null;
}

export function queryApp<E extends Element = HTMLElement>(selector: string, node?: Node | null): E | null {
  return appRoot(node).querySelector<E>(selector);
}

export function queryAppAll<E extends Element = HTMLElement>(selector: string, node?: Node | null): E[] {
  return Array.from(appRoot(node).querySelectorAll<E>(selector));
}

/** 루트 안에서 실제로 포커스를 가진 요소(Shadow DOM 이면 호스트가 아니라 그 안의 요소). */
export function activeAppElement(node?: Node | null): HTMLElement | null {
  const active = appRoot(node).activeElement;
  return active instanceof HTMLElement ? active : null;
}

/** 이벤트가 el 안에서 일어났는가 — composedPath 로 판정한다(Shadow DOM 경계에서 target 이 호스트로 바뀌어도 맞게). */
export function eventWithin(event: Event, el: Node | null | undefined): boolean {
  if (!el) { return false; }
  const path = typeof event.composedPath === "function" ? event.composedPath() : [];
  if (path.length > 0) { return path.includes(el); }
  return event.target instanceof Node && el.contains(event.target);
}

/** 이벤트가 처음 일어난 요소(Shadow DOM 안이면 그 안의 요소). */
export function eventOrigin(event: Event): EventTarget | null {
  const path = typeof event.composedPath === "function" ? event.composedPath() : [];
  return path[0] ?? event.target;
}

interface ComposedSelection {
  getComposedRanges?: (options?: { shadowRoots?: ShadowRoot[] } | ShadowRoot) => StaticRange[];
}
interface ShadowSelection {
  getSelection?: () => Selection | null;
}

/**
 * container 안의 현재 선택 범위. 브라우저마다 Shadow DOM 안의 선택을 돌려주는 방식이 달라서(Chrome 은
 * window.getSelection 이 호스트로 접히고 ShadowRoot.getSelection 을, 새 표준은 getComposedRanges 를 쓴다)
 * 차례로 시도하고, container 밖이면 null.
 */
export function selectionRangeIn(container: HTMLElement): Range | null {
  const root = container.getRootNode();
  const sel = window.getSelection();
  if (root instanceof ShadowRoot) {
    const composedSel = sel as unknown as ComposedSelection | null;
    if (sel && composedSel && typeof composedSel.getComposedRanges === "function" && !sel.isCollapsed) {
      const ranges = composedSel.getComposedRanges({ shadowRoots: [root] });
      const first = ranges[0];
      if (first && !first.collapsed && container.contains(first.startContainer)) {
        const range = document.createRange();
        range.setStart(first.startContainer, first.startOffset);
        range.setEnd(first.endContainer, first.endOffset);
        return range;
      }
    }
    const shadowSel = (root as ShadowRoot & ShadowSelection).getSelection?.();
    if (shadowSel && !shadowSel.isCollapsed && shadowSel.rangeCount > 0) {
      const range = shadowSel.getRangeAt(0);
      if (container.contains(range.commonAncestorContainer)) { return range; }
    }
  }
  if (!sel || sel.isCollapsed || sel.rangeCount === 0) { return null; }
  const range = sel.getRangeAt(0);
  return container.contains(range.commonAncestorContainer) ? range : null;
}

/** 선택을 지운다(Shadow DOM 쪽 선택까지). */
export function clearAppSelection(node?: Node | null): void {
  window.getSelection()?.removeAllRanges();
  const root = appRoot(node);
  if (root instanceof ShadowRoot) { (root as ShadowRoot & ShadowSelection).getSelection?.()?.removeAllRanges(); }
}
