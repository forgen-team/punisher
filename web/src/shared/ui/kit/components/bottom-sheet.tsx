
import * as React from "react";
import { createPortal } from "react-dom";
import { cn } from "../lib/utils";
import { useT } from "@/src/shared/i18n";
import { kitMessages } from "../lib/messages";
import { useBodyScrollLock } from "../hooks/useBodyScrollLock";
import { useFocusTrap } from "../hooks/useFocusTrap";
import { IconButton } from "./button";
import { portalRoot } from "@/src/shared/config/runtime";

export type BottomSheetDesktop = "dialog" | "side" | "sheet";

export interface BottomSheetProps {
  open: boolean;
  onClose: () => void;
  /** 시트 제목 — aria-labelledby 로 연결된다 */
  title: React.ReactNode;
  description?: React.ReactNode;
  children: React.ReactNode;
  /** 하단 고정 동작 영역 (safe-area 포함) */
  footer?: React.ReactNode;
  /**
   * md(768px) 이상에서의 모양.
   * dialog: 가운데 대화상자 · side: 오른쪽 패널 · sheet: 모바일과 동일한 하단 시트
   */
  desktop?: BottomSheetDesktop;
  /** 뒤 배경 클릭으로 닫기 (기본 true) */
  dismissible?: boolean;
  className?: string;
}

const DRAG_CLOSE_PX = 80;

const containerByDesktop: Record<BottomSheetDesktop, string> = {
  dialog: "md:items-center md:justify-center md:p-6",
  side: "md:items-stretch md:justify-end",
  sheet: "",
};

const panelByDesktop: Record<BottomSheetDesktop, string> = {
  dialog: "md:max-h-[min(85dvh,720px)] md:max-w-lg md:rounded-xl md:pb-0 md:animate-scale-in",
  side: "md:h-full md:max-h-none md:w-[400px] md:max-w-[90vw] md:rounded-none md:border-l md:border-t-0 md:pb-0 md:animate-sheet-right",
  sheet: "md:mx-auto md:max-w-xl",
};

/**
 * BottomSheet — 모바일 하단 시트, 데스크톱에서는 대화상자/측면 패널.
 * 포털 렌더, 포커스 가두기, Esc·배경 클릭 닫기, 손잡이 드래그로 닫기, safe-area 하단 여백.
 */
export function BottomSheet({
  open, onClose, title, description, children, footer, desktop = "dialog", dismissible = true, className,
}: BottomSheetProps) {
  const t = useT(kitMessages);
  const [mounted, setMounted] = React.useState(false);
  const [dragY, setDragY] = React.useState(0);
  const panelRef = React.useRef<HTMLDivElement>(null);
  const dragStart = React.useRef<number | null>(null);
  const titleId = React.useId();
  const descId = React.useId();

  React.useEffect(() => setMounted(true), []);
  React.useEffect(() => {
    if (!open) setDragY(0);
  }, [open]);
  useBodyScrollLock(open);
  useFocusTrap(panelRef, open && mounted, onClose);

  if (!open || !mounted) return null;

  function handlePointerDown(event: React.PointerEvent<HTMLDivElement>) {
    dragStart.current = event.clientY;
    event.currentTarget.setPointerCapture(event.pointerId);
  }
  function handlePointerMove(event: React.PointerEvent<HTMLDivElement>) {
    if (dragStart.current === null) return;
    setDragY(Math.max(0, event.clientY - dragStart.current));
  }
  function handlePointerUp() {
    if (dragStart.current === null) return;
    dragStart.current = null;
    if (dragY > DRAG_CLOSE_PX) onClose();
    else setDragY(0);
  }

  return createPortal(
    <div
      className={cn(
        "fixed inset-0 flex items-end justify-center",
        desktop === "dialog" ? "z-modal" : "z-sheet",
        containerByDesktop[desktop]
      )}
    >
      <div
        aria-hidden="true"
        className="absolute inset-0 animate-fade-in bg-overlay motion-reduce:animate-none"
        onClick={dismissible ? onClose : undefined}
      />
      <div
        ref={panelRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        aria-describedby={description ? descId : undefined}
        tabIndex={-1}
        style={dragY ? { transform: `translateY(${dragY}px)` } : undefined}
        className={cn(
          "relative flex max-h-[85dvh] w-full flex-col overflow-hidden rounded-t-xl border-t border-line",
          "bg-canvas-elevated text-fg shadow-overlay outline-none pb-safe-bottom",
          "animate-sheet-up motion-reduce:animate-none",
          panelByDesktop[desktop],
          className
        )}
      >
        <div
          className={cn("flex shrink-0 touch-none justify-center pb-1 pt-2", desktop !== "sheet" && "md:hidden")}
          onPointerDown={handlePointerDown}
          onPointerMove={handlePointerMove}
          onPointerUp={handlePointerUp}
          onPointerCancel={handlePointerUp}
          aria-hidden="true"
        >
          <span className="h-1 w-9 rounded-full bg-line-strong" />
        </div>
        <div className="flex shrink-0 items-start gap-2 px-4 pb-2 md:pt-3">
          <div className="min-w-0 flex-1 py-2">
            <h2 id={titleId} className="text-base font-semibold text-fg">{title}</h2>
            {description ? <p id={descId} className="mt-0.5 text-sm text-fg-secondary">{description}</p> : null}
          </div>
          <IconButton aria-label={t.close} onClick={onClose} className="-mr-2">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" aria-hidden="true">
              <path d="M18 6 6 18M6 6l12 12" />
            </svg>
          </IconButton>
        </div>
        <div className="min-h-0 flex-1 overflow-y-auto overscroll-contain px-4 pb-6">{children}</div>
        {footer ? <div className="flex shrink-0 justify-end gap-2 border-t border-line px-4 py-3">{footer}</div> : null}
      </div>
    </div>,
    portalRoot()
  );
}
