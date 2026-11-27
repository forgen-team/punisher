import { useEffect, useRef, useState, type RefObject } from "react";
import { MessageCircle, X } from "lucide-react";
import { IconButton, useMediaQuery } from "@forgen-team/ui";
import { cn, focusRing } from "@/src/shared/lib/cn";
import { eventWithin } from "@/src/shared/lib/dom-root";
import { ReadingOverlay, toOverlayAnchor } from "@/src/shared/ui/punisher";
import { useT } from "@/src/shared/i18n";
import { useReadingThreads } from "./ReadingThreadsProvider";
import { SelectionToolbar, SentenceTap } from "./SelectionActions";
import { ThreadBody, ThreadView } from "./ThreadView";
import { findQuoteRange, paintHighlight, scrollRangeBelowTopBar } from "./lib/textRange";
import { countPassageThreads, findThread, isLiveThread, type PassageThreads, type ReadingThread } from "./lib/threads";
import { messages } from "./i18n";

const THREAD_HIGHLIGHT = "punisher-thread";
/** 오른쪽 보조 열이 보이는 폭(LectureLayout xl) — 이 폭부터 스레드는 패널 '질문' 탭에서 펼친다. */
export const THREAD_PANEL_QUERY = "(min-width: 1280px)";

/** 패널이 없는 폭: 모바일은 절반 높이 시트, 데스크톱(md~xl)은 구절 옆 팝오버. */
function ThreadOverlay() {
  const t = useT(messages);
  const ctx = useReadingThreads();
  const wide = useMediaQuery(THREAD_PANEL_QUERY);
  if (!ctx) { return null; }
  const { focus } = ctx;
  const rect = focus?.kind === "draft" ? focus.passage.rect : (focus?.rect ?? null);
  return (
    <ReadingOverlay
      open={focus !== null && !wide}
      onClose={ctx.close}
      anchor={rect}
      title={t.threadTitle}
      description={t.threadDescription}
      sheetSize="half"
    >
      {/* 스레드마다 새로 그려 이어 묻기 입력·요청 상태가 다른 스레드로 새지 않게 한다. */}
      {focus && <ThreadView key={focusKey(focus)} focus={focus} threads={ctx.threads} />}
    </ReadingOverlay>
  );
}

function focusKey(focus: NonNullable<ReturnType<typeof useReadingThreads>>["focus"]): string {
  return focus?.kind === "thread" ? focus.rootId : "draft";
}

/** 속성 선택자 값 이스케이프(CSS.escape 가 없는 환경 대비). */
function cssEscape(value: string): string {
  return typeof CSS !== "undefined" && typeof CSS.escape === "function" ? CSS.escape(value) : value.replace(/["\\]/g, "\\$&");
}

function anchorElement(container: HTMLElement, anchor: string): Element | null {
  if (!anchor) { return null; }
  return container.querySelector(`[data-lecture-anchor="${cssEscape(anchor)}"]`);
}

/** 대화가 달린 구절에 밑줄(본문 DOM 은 그대로). */
function ThreadHighlights({ containerRef }: { containerRef: RefObject<HTMLElement> }) {
  const ctx = useReadingThreads();
  const passages = ctx?.passages;
  useEffect(() => {
    const container = containerRef.current;
    if (!container || !passages) { return undefined; }
    const frame = requestAnimationFrame(() => {
      const ranges = passages
        .map((p) => {
          const el = anchorElement(container, p.anchor);
          return el ? findQuoteRange(el, p.quote) : null;
        })
        .filter((r): r is Range => r !== null);
      paintHighlight(THREAD_HIGHLIGHT, ranges);
    });
    return () => cancelAnimationFrame(frame);
  }, [containerRef, passages]);
  useEffect(() => () => paintHighlight(THREAD_HIGHLIGHT, []), []);
  return null;
}

/**
 * 강의 본문 읽기 도우미 — 데스크톱 드래그 도구 막대, 모바일 문장 탭 액션 바, 스레드 시트/팝오버, 구절 밑줄.
 * ReadingThreadsProvider 안에서 쓴다.
 */
export function ReadingAssist({ containerRef }: { containerRef: RefObject<HTMLElement> }) {
  const coarse = useMediaQuery("(pointer: coarse)");
  return (
    <>
      {coarse ? <SentenceTap containerRef={containerRef} /> : <SelectionToolbar containerRef={containerRef} />}
      <ThreadOverlay />
      <ThreadHighlights containerRef={containerRef} />
    </>
  );
}

/** 본문 블록 끝의 '대화 n' — 누르면 그 구절의 최신 스레드를 연다. */
export function PassageMarkers({ anchor }: { anchor: string }) {
  const t = useT(messages);
  const ctx = useReadingThreads();
  if (!ctx || !anchor) { return null; }
  // 실패만 남은 구절(대화 0)은 표시하지 않는다 — 질문 탭 목록에서 다시 시도할 수 있다.
  const here = ctx.passages.filter((p) => p.anchor === anchor && p.exchanges > 0);
  if (here.length === 0) { return null; }
  return (
    <div className="mt-1 flex flex-wrap gap-1" data-no-sentence-tap="">
      {here.map((p) => (
        <button
          key={p.key}
          type="button"
          title={p.quote}
          aria-label={t.openThread(p.exchanges, p.quote)}
          onClick={(event) => {
            const newest = p.threads.find((thread) => isLiveThread(thread)) ?? p.threads[0];
            if (!newest) { return; }
            // 모바일 절반 시트에서도 대화의 시작 구절이 시트 위에 보이게 올린다.
            const block = event.currentTarget.closest("[data-lecture-anchor]");
            const range = block ? findQuoteRange(block, p.quote) : null;
            if (range) { scrollRangeBelowTopBar(range); }
            ctx.open(newest.root.id, toOverlayAnchor(event.currentTarget.getBoundingClientRect()));
          }}
          className={cn(
            // 시각 24px + 위아래 확장 = 44px 터치 영역
            "relative inline-flex h-6 items-center gap-1 rounded-full border border-line bg-canvas-elevated px-2 text-xs text-accent-text",
            "before:absolute before:inset-x-0 before:-inset-y-2.5 before:content-[''] hover:bg-canvas-hover",
            focusRing,
          )}
        >
          <MessageCircle className="size-3.5" aria-hidden="true" />
          {t.conversations(p.exchanges)}
        </button>
      ))}
    </div>
  );
}

/** 강의 툴바 질문 아이콘 위 개수(구절 스레드 루트 수 — 실패만 남은 루트·예전 방식 질문·이어 묻기는 세지 않는다). 없으면 그리지 않는다. */
export function ThreadCountBadge() {
  const ctx = useReadingThreads();
  const count = ctx ? countPassageThreads(ctx.passages) : 0;
  if (count === 0) { return null; }
  return (
    <span aria-hidden="true" className="pointer-events-none absolute right-1 top-1 inline-flex h-4 min-w-4 items-center justify-center rounded-full bg-accent-solid px-1 text-[10px] font-semibold leading-none text-accent-on">
      {count > 99 ? "99+" : count}
    </span>
  );
}

function threadPreview(thread: ReadingThread, t: typeof messages.ko): string {
  const last = thread.replies[thread.replies.length - 1] ?? thread.root;
  const isReply = last !== thread.root;
  if (last.mode === "easier") { return isReply ? t.moreEasier : t.easier; }
  if (last.mode === "deeper") { return isReply ? t.moreDeeper : t.deeper; }
  return last.question || thread.root.question;
}

/** 패널(≥1280px)에서 열린 스레드를 닫는 X — 본문으로 돌아가며 칠해 둔 구절을 지운다. */
function CloseFocus({ onClose }: { onClose: () => void }) {
  const t = useT(messages);
  return (
    <IconButton size="sm" aria-label={t.closeThread} onClick={onClose} className="shrink-0" icon={<X aria-hidden="true" />} />
  );
}

interface PassageGroupProps {
  passage: PassageThreads;
  expandedId: string | null;
  onExpand: (id: string) => void;
  /** 패널에서 지금 열린 스레드 id — 그 스레드에 닫기 버튼을 단다 */
  focusId: string | null;
  onCloseFocus: () => void;
}

function PassageGroup({ passage, expandedId, onExpand, focusId, onCloseFocus }: PassageGroupProps) {
  const t = useT(messages);
  return (
    <li className="flex flex-col gap-2 border-b border-line py-3 last:border-b-0">
      <div className="flex items-start gap-2">
        <blockquote className="line-clamp-3 min-w-0 flex-1 border-l-2 border-line-strong pl-3 text-sm text-fg-secondary">{passage.quote}</blockquote>
        {focusId !== null && passage.threads.some((th) => th.root.id === focusId) && <CloseFocus onClose={onCloseFocus} />}
      </div>
      <ul role="list" className="flex flex-col gap-2">
        {passage.threads.map((thread) =>
          thread.root.id === expandedId ? (
            <li key={thread.root.id} data-thread-id={thread.root.id}>
              <ThreadBody thread={thread} showQuote={false} />
            </li>
          ) : (
            <li key={thread.root.id}>
              <button
                type="button"
                aria-label={`${t.expandThread}: ${threadPreview(thread, t)}`}
                onClick={() => onExpand(thread.root.id)}
                className={cn("flex min-h-touch w-full items-center gap-2 rounded-md px-1 text-left text-sm text-fg hover:bg-canvas-hover", focusRing)}
              >
                <MessageCircle className="size-4 shrink-0 text-fg-muted" aria-hidden="true" />
                <span className="min-w-0 flex-1 truncate">{threadPreview(thread, t)}</span>
                <span className="shrink-0 text-xs text-fg-muted">{t.conversations(1 + thread.replies.length)}</span>
              </button>
            </li>
          ),
        )}
      </ul>
    </li>
  );
}

/**
 * 질문 탭의 구절별 스레드(데스크톱 패널·모바일 질문 시트 공용) — 열린(또는 가장 최근) 스레드만 펼친다.
 * 패널 폭에서는 본문에서 고른 새 스레드(draft)도 맨 위에 펼친다. 패널 폭에서 열린 스레드는
 * X · Esc · 패널 바깥 누르기로 닫고, 새 스레드의 답이 오면(보내기 성공) 펼친 채로 닫는다.
 */
export function PassageThreadList() {
  const t = useT(messages);
  const ctx = useReadingThreads();
  const wide = useMediaQuery(THREAD_PANEL_QUERY);
  const sectionRef = useRef<HTMLElement>(null);
  const listRef = useRef<HTMLDivElement>(null);
  const frame = useRef(0);
  const focus = ctx?.focus ?? null;
  const focusId = focus?.kind === "thread" ? focus.rootId : null;
  const [expanded, setExpanded] = useState<string | null>(null);
  const prevKind = useRef<"draft" | "thread" | null>(null);
  const close = ctx?.close;
  const threads = ctx?.threads;
  const openSeq = ctx?.openSeq ?? 0;
  const active = wide && focus !== null;

  useEffect(() => {
    if (focusId) { setExpanded(focusId); }
  }, [focusId]);

  // 패널 폭: draft 의 답이 성공으로 오면(스레드로 바뀌면) 펼친 채로 열린 상태를 닫는다(실패면 다시 시도하게 둔다).
  useEffect(() => {
    const wasDraft = prevKind.current === "draft";
    prevKind.current = focus?.kind ?? null;
    if (!wide || !wasDraft || !focusId || !close || !threads) { return; }
    const thread = findThread(threads, focusId);
    if (thread && isLiveThread(thread)) { close(); }
  }, [wide, focus, focusId, threads, close]);

  // 새로 연 스레드(또는 draft)로 스크롤 — 닫혀도 예약한 스크롤은 그대로, 다음 열기·언마운트 때 취소한다.
  const hasFocus = focus !== null;
  useEffect(() => {
    if (!wide || !hasFocus) { return; }
    cancelAnimationFrame(frame.current);
    frame.current = requestAnimationFrame(() => {
      const target = focusId ? listRef.current?.querySelector(`[data-thread-id="${cssEscape(focusId)}"]`) : listRef.current?.firstElementChild;
      target?.scrollIntoView({ block: "nearest", behavior: "smooth" });
    });
  }, [wide, hasFocus, focusId, openSeq]);
  useEffect(() => () => cancelAnimationFrame(frame.current), []);

  // 패널 폭: Esc · 패널 바깥 누르기로 닫는다.
  useEffect(() => {
    if (!active || !close) { return undefined; }
    const onKey = (event: KeyboardEvent) => {
      if (event.key === "Escape" && !event.defaultPrevented) { close(); }
    };
    const onPointer = (event: PointerEvent) => {
      const section = sectionRef.current;
      if (section && !eventWithin(event, section)) { close(); }
    };
    document.addEventListener("keydown", onKey);
    document.addEventListener("pointerdown", onPointer);
    return () => {
      document.removeEventListener("keydown", onKey);
      document.removeEventListener("pointerdown", onPointer);
    };
  }, [active, close]);

  if (!ctx) { return null; }
  const passages = ctx.passages;
  const fallback = passages[0]?.threads[0]?.root.id ?? null;
  const expandedId = expanded ?? fallback;
  const showDraft = wide && focus?.kind === "draft";
  if (passages.length === 0 && !showDraft) { return null; }

  return (
    <section ref={sectionRef} aria-label={t.passageThreads} className="flex flex-col">
      <div ref={listRef} className="flex flex-col">
        {showDraft && focus && (
          <div className="flex items-start gap-2 border-b border-line py-3">
            <div className="min-w-0 flex-1">
              <ThreadView key={focusKey(focus)} focus={focus} threads={ctx.threads} />
            </div>
            <CloseFocus onClose={ctx.close} />
          </div>
        )}
        <ul role="list">
          {passages.map((p) => (
            <PassageGroup key={p.key} passage={p} expandedId={expandedId} onExpand={setExpanded} focusId={wide ? focusId : null} onCloseFocus={ctx.close} />
          ))}
        </ul>
      </div>
    </section>
  );
}
