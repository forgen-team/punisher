import type { RefObject } from "react";
import { createPortal } from "react-dom";
import { MessageCirclePlus } from "lucide-react";
import { cn, focusRing, useMediaQuery } from "@/src/shared/ui/kit";
import { useTextSelection } from "@/src/features/punisher/reading-assist";
import { INQUIRY_MAX_QUOTE_CHARS } from "@/src/shared/api/punisher";
import { useT } from "@/src/shared/i18n";
import { messages } from "../i18n";

const BUTTON_GAP = 10;
const MIN_CHARS = 2;
const TOP_CLEARANCE = 112;

/** 해설에서 고른 구절 옆 '여기 더 자세히' — 누르면 그 구절을 인용으로 붙여 이어서 묻기로 간다. */
export function QuoteAsk({ containerRef, onQuote }: { containerRef: RefObject<HTMLElement>; onQuote: (text: string) => void }) {
  const t = useT(messages);
  const coarse = useMediaQuery("(pointer: coarse)");
  const selection = useTextSelection(containerRef, true);
  if (!selection || Array.from(selection.text).length < MIN_CHARS) { return null; }
  return createPortal(
    <div
      className="absolute z-panel"
      style={{
        // 선택이 화면 위쪽이면 고정 상단 바에 가리지 않게 아래로 내린다.
        top: Math.max(TOP_CLEARANCE, coarse ? selection.rect.bottom + BUTTON_GAP : selection.rect.top - BUTTON_GAP - 40) + window.scrollY,
        left: Math.max(8, Math.min(selection.rect.left, window.innerWidth - 180)) + window.scrollX,
      }}
    >
      <button
        type="button"
        onPointerDown={(event) => event.preventDefault()}
        onClick={() => {
          onQuote(Array.from(selection.text).slice(0, INQUIRY_MAX_QUOTE_CHARS).join(""));
          window.getSelection()?.removeAllRanges();
        }}
        className={cn(
          "inline-flex h-10 items-center gap-1.5 rounded-full border border-line bg-canvas-elevated px-3.5 text-sm font-medium",
          "text-fg shadow-overlay hover:bg-canvas-hover",
          focusRing,
        )}
      >
        <MessageCirclePlus className="size-4 text-accent-text" aria-hidden="true" />
        {t.quoteAsk}
      </button>
    </div>,
    document.body,
  );
}
