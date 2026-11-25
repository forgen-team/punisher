
import { useEffect, useState, type RefObject } from "react";
import { toOverlayAnchor, type OverlayAnchor } from "@/src/shared/ui/punisher";
import { selectionRangeIn } from "@/src/shared/lib/dom-root";

export interface TextSelectionSnapshot {
  text: string;
  /** 선택이 시작된 블록/섹션의 진도 앵커(data-lecture-anchor) */
  anchor: string;
  /** 선택 범위의 뷰포트 좌표(잡은 순간 기준) */
  rect: OverlayAnchor;
}

function closestAnchor(node: Node | null): string {
  const el = node instanceof Element ? node : node?.parentElement;
  return el?.closest<HTMLElement>("[data-lecture-anchor]")?.dataset.lectureAnchor ?? "";
}

/** Shadow DOM 안 선택은 브라우저마다 얻는 방법이 달라 selectionRangeIn 이 맞춰 준다(점검 26). */
function readSelection(container: HTMLElement): TextSelectionSnapshot | null {
  const range = selectionRangeIn(container);
  if (!range) { return null; }
  const text = range.toString().trim();
  if (!text) { return null; }
  const rect = range.getBoundingClientRect();
  if (rect.width === 0 && rect.height === 0) { return null; }
  return { text, anchor: closestAnchor(range.startContainer), rect: toOverlayAnchor(rect) };
}

/**
 * 컨테이너 안의 텍스트 선택(마우스 드래그·모바일 길게 눌러 선택)을 추적한다.
 * selectionchange 는 드래그 중에도 계속 오므로 한 프레임 모아서 읽는다.
 * 선택을 건드리지 않으므로 기본 복사·공유 메뉴는 그대로 동작한다.
 */
export function useTextSelection(containerRef: RefObject<HTMLElement>, enabled: boolean): TextSelectionSnapshot | null {
  const [snapshot, setSnapshot] = useState<TextSelectionSnapshot | null>(null);

  useEffect(() => {
    if (!enabled) {
      setSnapshot(null);
      return;
    }
    let frame = 0;
    function update() {
      cancelAnimationFrame(frame);
      frame = requestAnimationFrame(() => {
        const container = containerRef.current;
        const next = container ? readSelection(container) : null;
        setSnapshot((prev) => {
          if (!next) { return prev ? null : prev; }
          if (prev && prev.text === next.text && prev.rect.top === next.rect.top && prev.rect.left === next.rect.left) { return prev; }
          return next;
        });
      });
    }
    // 스크롤하면 뷰포트 좌표가 바뀌므로 다시 읽는다(버튼은 문서 좌표로 붙지만 rect 는 최신이어야 한다).
    document.addEventListener("selectionchange", update);
    window.addEventListener("scroll", update, { passive: true });
    return () => {
      cancelAnimationFrame(frame);
      document.removeEventListener("selectionchange", update);
      window.removeEventListener("scroll", update);
    };
  }, [containerRef, enabled]);

  return snapshot;
}
