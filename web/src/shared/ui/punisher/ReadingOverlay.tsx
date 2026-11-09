
import { useEffect, useId, useLayoutEffect, useRef, useState, type ReactNode } from "react";
import { createPortal } from "react-dom";
import { IconButton, cn, useFocusTrap, useMediaQuery } from "@/src/shared/ui/kit";

/** 읽기 위치를 기준으로 붙일 사각형(문서 좌표가 아니라 getBoundingClientRect 값). */
export interface OverlayAnchor {
  top: number;
  bottom: number;
  left: number;
  right: number;
}

export function toOverlayAnchor(rect: DOMRect): OverlayAnchor {
  return { top: rect.top, bottom: rect.bottom, left: rect.left, right: rect.right };
}

interface ReadingOverlayProps {
  open: boolean;
  onClose: () => void;
  /** 데스크톱 팝오버를 붙일 위치(열 때의 뷰포트 좌표) */
  anchor: OverlayAnchor | null;
  title: ReactNode;
  description?: ReactNode;
  children: ReactNode;
  footer?: ReactNode;
}

const POPOVER_WIDTH = 380;
const GAP = 8;
const MARGIN = 12;

/**
 * 강의 읽기 중 보조 설명 — 본문을 밀어내거나 스크롤 위치를 바꾸지 않는 오버레이.
 * 데스크톱(md+): 대상 옆에 붙는 팝오버(문서 좌표, 본문과 함께 스크롤). 모바일: 하단 시트.
 * BottomSheet 와 달리 body 스크롤 잠금(position:fixed)을 쓰지 않아 열고 닫아도 window.scrollY 가 그대로다.
 * 포커스 이동·복귀는 preventScroll 로만 한다(useFocusTrap).
 */
export function ReadingOverlay({ open, onClose, anchor, title, description, children, footer }: ReadingOverlayProps) {
  const desktop = useMediaQuery("(min-width: 768px)");
  const panelRef = useRef<HTMLDivElement>(null);
  const [mounted, setMounted] = useState(false);
  const [position, setPosition] = useState<{ top: number; left: number } | null>(null);
  const titleId = useId();

  useEffect(() => setMounted(true), []);
  useFocusTrap(panelRef, open && mounted, onClose);

  // 데스크톱: 대상 아래(공간이 없으면 위)에 붙인다. 문서 좌표로 바꿔 두어 스크롤해도 대상과 함께 움직인다.
  useLayoutEffect(() => {
    if (!open || !desktop || !anchor) {
      setPosition(null);
      return;
    }
    const panel = panelRef.current;
    const height = panel?.offsetHeight ?? 240;
    const viewportW = window.innerWidth;
    const viewportH = window.innerHeight;
    const left = Math.min(Math.max(MARGIN, anchor.left), viewportW - POPOVER_WIDTH - MARGIN);
    const below = anchor.bottom + GAP;
    const top = below + height > viewportH - MARGIN && anchor.top - GAP - height > MARGIN ? anchor.top - GAP - height : below;
    setPosition({ top: top + window.scrollY, left: left + window.scrollX });
  }, [open, desktop, anchor]);

  // 바깥을 누르면 닫는다(데스크톱 팝오버 — 시트는 배경 레이어가 받는다).
  useEffect(() => {
    if (!open || !desktop) { return; }
    function onPointerDown(event: PointerEvent) {
      if (!panelRef.current?.contains(event.target as Node)) { onClose(); }
    }
    document.addEventListener("pointerdown", onPointerDown);
    return () => document.removeEventListener("pointerdown", onPointerDown);
  }, [open, desktop, onClose]);

  if (!open || !mounted) { return null; }

  const header = (
    <div className="flex shrink-0 items-start gap-2 px-4 pb-1 pt-3">
      <div className="min-w-0 flex-1">
        <h2 id={titleId} className="text-base font-semibold text-fg">{title}</h2>
        {description ? <p className="mt-0.5 text-xs text-fg-muted">{description}</p> : null}
      </div>
      <IconButton aria-label="닫기" onClick={onClose} className="-mr-2 -mt-1">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" aria-hidden="true">
          <path d="M18 6 6 18M6 6l12 12" />
        </svg>
      </IconButton>
    </div>
  );
  const body = <div className="min-h-0 flex-1 overflow-y-auto overscroll-contain px-4 pb-4 pt-2">{children}</div>;
  const foot = footer ? <div className="flex shrink-0 justify-end gap-2 border-t border-line px-4 py-2">{footer}</div> : null;

  if (desktop) {
    return createPortal(
      <div
        ref={panelRef}
        role="dialog"
        aria-modal="false"
        aria-labelledby={titleId}
        tabIndex={-1}
        style={{ top: position?.top ?? -9999, left: position?.left ?? -9999, width: POPOVER_WIDTH }}
        className={cn(
          "absolute z-panel flex max-h-[60vh] flex-col overflow-hidden rounded-lg border border-line",
          "bg-canvas-elevated text-fg shadow-overlay outline-none animate-fade-in motion-reduce:animate-none",
        )}
      >
        {header}
        {body}
        {foot}
      </div>,
      document.body,
    );
  }

  return createPortal(
    <div className="fixed inset-0 z-sheet flex items-end">
      {/* touch-none: 배경을 끌어도 뒤의 본문이 스크롤되지 않게(스크롤 잠금 없이) */}
      <div aria-hidden="true" className="absolute inset-0 touch-none bg-overlay animate-fade-in motion-reduce:animate-none" onClick={onClose} />
      <div
        ref={panelRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        tabIndex={-1}
        className={cn(
          "relative flex max-h-[70dvh] w-full flex-col overflow-hidden rounded-t-xl border-t border-line",
          "bg-canvas-elevated text-fg shadow-overlay outline-none pb-safe-bottom animate-sheet-up motion-reduce:animate-none",
        )}
      >
        <div className="flex shrink-0 justify-center pt-2" aria-hidden="true">
          <span className="h-1 w-9 rounded-full bg-line-strong" />
        </div>
        {header}
        {body}
        {foot}
      </div>
    </div>,
    document.body,
  );
}
