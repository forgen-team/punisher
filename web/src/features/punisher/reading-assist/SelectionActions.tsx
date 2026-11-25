import { useEffect, useRef, useState, type RefObject } from "react";
import { createPortal } from "react-dom";
import { HelpCircle, Layers, Sparkles, StickyNote, X } from "lucide-react";
import { cn, focusRing, useMediaQuery } from "@/src/shared/ui/kit";
import { SELECTION_MAX_CHARS, SELECTION_MIN_CHARS, type ThreadMode } from "@/src/shared/api/punisher";
import { toOverlayAnchor } from "@/src/shared/ui/punisher";
import { portalRoot } from "@/src/shared/config/runtime";
import { clearAppSelection, selectionRangeIn } from "@/src/shared/lib/dom-root";
import { useT } from "@/src/shared/i18n";
import { useReadingThreads, type ThreadPassage } from "./ReadingThreadsProvider";
import { useTextSelection } from "./useTextSelection";
import { isIgnoredTapTarget, paintHighlight, scrollRangeBelowTopBar, sentenceAtPoint, supportsHighlight, type TappedSentence } from "./lib/textRange";
import { messages } from "./i18n";

const BUTTON_GAP = 10;
/** 본문을 고른 채로 도구 막대로 옮겨 가는 키(aria-keyshortcuts 값과 같다). */
export const TOOLBAR_SHORTCUT = "Alt+Enter";
const SENTENCE_HIGHLIGHT = "punisher-sentence";
export type PassageAction = ThreadMode | "memo";

const ACTIONS: Array<{ key: PassageAction; icon: typeof Sparkles; label: "easier" | "deeper" | "question" | "memo" }> = [
  { key: "easier", icon: Sparkles, label: "easier" },
  { key: "deeper", icon: Layers, label: "deeper" },
  { key: "question", icon: HelpCircle, label: "question" },
  { key: "memo", icon: StickyNote, label: "memo" },
];

const actionButton = cn(
  "inline-flex min-h-touch min-w-touch items-center justify-center gap-1.5 rounded-full px-3 text-sm font-medium text-fg",
  "hover:bg-canvas-hover disabled:opacity-60",
  focusRing,
);

/** 쉽게 · 깊게 · 질문 · 메모 — 데스크톱 도구 막대와 모바일 액션 바가 같이 쓴다. */
function ActionButtons({ onAction, disabled }: { onAction: (action: PassageAction) => void; disabled?: boolean }) {
  const t = useT(messages);
  return (
    <>
      {ACTIONS.map(({ key, icon: Icon, label }) => (
        <button
          key={key}
          type="button"
          disabled={disabled}
          // 마우스 버튼을 누르는 순간 선택이 풀리지 않게 한다(데스크톱).
          onPointerDown={(event) => event.preventDefault()}
          onClick={() => onAction(key)}
          className={actionButton}
        >
          <Icon className="size-4 text-accent-text" aria-hidden="true" />
          {t[label]}
        </button>
      ))}
    </>
  );
}

/** 메모 저장 뒤 잠깐 보여 줄 한 줄. */
function useMemoStatus() {
  const [status, setStatus] = useState<"saved" | "failed" | null>(null);
  useEffect(() => {
    if (!status) { return undefined; }
    const timer = window.setTimeout(() => setStatus(null), 1800);
    return () => window.clearTimeout(timer);
  }, [status]);
  return [status, setStatus] as const;
}

function MemoStatus({ status }: { status: "saved" | "failed" | null }) {
  const t = useT(messages);
  return (
    <span role="status" className="px-2 text-sm text-fg-secondary empty:hidden">
      {status === "saved" ? t.memoSaved : status === "failed" ? t.memoFailed : ""}
    </span>
  );
}

/**
 * 데스크톱: 드래그 선택 위에 뜨는 도구 막대(쉽게·깊게·질문·메모).
 * 화살표 이동이 없는 버튼 묶음이라 role=group(Tab 으로 이동). 글을 고른 채 Alt+Enter 로 첫 버튼에 포커스를 옮긴다.
 */
export function SelectionToolbar({ containerRef }: { containerRef: RefObject<HTMLElement> }) {
  const t = useT(messages);
  const ctx = useReadingThreads();
  const selection = useTextSelection(containerRef, true);
  const [memo, setMemo] = useMemoStatus();
  const groupRef = useRef<HTMLDivElement>(null);
  const length = selection ? Array.from(selection.text).length : 0;
  const shown = selection !== null && length >= SELECTION_MIN_CHARS;

  useEffect(() => {
    if (!shown) { return undefined; }
    const onKey = (event: KeyboardEvent) => {
      if (!event.altKey || event.key !== "Enter") { return; }
      const first = groupRef.current?.querySelector<HTMLElement>("button:not(:disabled)");
      if (!first) { return; }
      event.preventDefault();
      first.focus();
    };
    document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
  }, [shown]);

  if (!ctx) { return null; }

  const tooLong = length > SELECTION_MAX_CHARS;
  if (!selection || !shown) {
    return memo ? createPortal(<div className="fixed bottom-6 left-1/2 z-panel -translate-x-1/2 rounded-full border border-line bg-canvas-elevated px-3 py-2 shadow-overlay"><MemoStatus status={memo} /></div>, portalRoot()) : null;
  }

  const passage: ThreadPassage = { anchor: selection.anchor, quote: selection.text, rect: selection.rect };
  function act(action: PassageAction) {
    if (!ctx) { return; }
    clearAppSelection(groupRef.current);
    if (action === "memo") {
      ctx.saveMemo(passage).then(() => setMemo("saved"), () => setMemo("failed"));
      return;
    }
    ctx.start(passage, action);
  }

  return createPortal(
    <div
      ref={groupRef}
      role="group"
      aria-label={t.actionsLabel}
      aria-keyshortcuts={TOOLBAR_SHORTCUT}
      className="absolute z-panel flex items-center gap-0.5 rounded-full border border-line bg-canvas-elevated p-0.5 shadow-overlay"
      style={{
        top: selection.rect.top - BUTTON_GAP - 46 + window.scrollY,
        left: Math.max(8, Math.min(selection.rect.left, window.innerWidth - 360)) + window.scrollX,
      }}
      title={tooLong ? t.maxChars(SELECTION_MAX_CHARS) : undefined}
    >
      {tooLong ? <span className="px-3 py-2 text-sm text-fg-secondary">{t.tooLong(SELECTION_MAX_CHARS)}</span> : <ActionButtons onAction={act} />}
    </div>,
    portalRoot(),
  );
}


/**
 * 모바일(pointer: coarse): 본문 문장을 탭하면 그 문장을 칠하고 아래 액션 바(쉽게·깊게·질문·메모·닫기)를 띄운다.
 * 링크·용어·버튼·코드 블록 탭, 길게 눌러 선택한 경우는 그대로 둔다.
 */
export function SentenceTap({ containerRef }: { containerRef: RefObject<HTMLElement> }) {
  const t = useT(messages);
  const ctx = useReadingThreads();
  const coarse = useMediaQuery("(pointer: coarse)");
  const [tapped, setTapped] = useState<TappedSentence | null>(null);
  // 액션을 고른 뒤에도 스레드 시트가 열려 있는 동안은 문장을 칠해 둔다.
  const [held, setHeld] = useState<Range | null>(null);
  const [memo, setMemo] = useMemoStatus();
  const tappedRef = useRef<TappedSentence | null>(null);
  tappedRef.current = tapped;
  const focusOpen = ctx?.focus !== null && ctx?.focus !== undefined;
  const [highlightable] = useState(supportsHighlight);

  useEffect(() => {
    const container = containerRef.current;
    if (!coarse || !container) { return undefined; }
    function onClick(event: MouseEvent) {
      const host = containerRef.current;
      if (!host || isIgnoredTapTarget(event.target)) { return; }
      if (selectionRangeIn(host)) { return; }
      const next = sentenceAtPoint(host, event.clientX, event.clientY);
      const prev = tappedRef.current;
      if (!next || (prev && prev.text === next.text && prev.anchor === next.anchor)) {
        setTapped(null);
        return;
      }
      setTapped(next);
    }
    container.addEventListener("click", onClick);
    return () => container.removeEventListener("click", onClick);
  }, [coarse, containerRef]);

  useEffect(() => {
    if (!focusOpen) { setHeld(null); }
  }, [focusOpen]);

  useEffect(() => {
    const range = tapped?.range ?? held;
    paintHighlight(SENTENCE_HIGHLIGHT, range ? [range] : []);
  }, [tapped, held]);
  useEffect(() => () => paintHighlight(SENTENCE_HIGHLIGHT, []), []);

  useEffect(() => {
    if (!tapped) { return undefined; }
    const onKey = (event: KeyboardEvent) => { if (event.key === "Escape") { setTapped(null); } };
    document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
  }, [tapped]);

  if (!ctx || !coarse || (!tapped && !memo)) { return null; }

  function act(action: PassageAction) {
    if (!ctx || !tapped) { return; }
    const passage: ThreadPassage = { anchor: tapped.anchor, quote: tapped.text, rect: toOverlayAnchor(tapped.range.getBoundingClientRect()) };
    setTapped(null);
    if (action === "memo") {
      ctx.saveMemo(passage).then(() => setMemo("saved"), () => setMemo("failed"));
      return;
    }
    setHeld(tapped.range);
    scrollRangeBelowTopBar(tapped.range);
    ctx.start(passage, action);
  }

  return createPortal(
    <div
      role="group"
      aria-label={t.actionsLabel}
      className="fixed inset-x-0 bottom-[var(--app-bottom-offset,0px)] z-panel flex flex-col border-t border-line bg-canvas-elevated px-2 py-1 pb-safe-bottom shadow-overlay"
    >
      {tapped ? (
        <>
          {/* 칠하기(CSS Highlight)를 못 쓰는 브라우저: 어느 문장을 골랐는지 한 줄로 보여 준다. */}
          {!highlightable && <p className="truncate px-2 pt-1 text-xs text-fg-secondary">{tapped.text}</p>}
          <div className="flex items-center justify-between gap-1">
            <div className="flex min-w-0 flex-1 items-center justify-around">
              <ActionButtons onAction={act} />
            </div>
            <button type="button" aria-label={t.close} onClick={() => setTapped(null)} className={cn(actionButton, "px-0")}>
              <X className="size-5" aria-hidden="true" />
            </button>
          </div>
        </>
      ) : (
        <MemoStatus status={memo} />
      )}
    </div>,
    portalRoot(),
  );
}

