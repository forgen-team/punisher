import * as React from "react";
import { createPortal } from "react-dom";
import { portalRoot } from "@/src/shared/config/runtime";
import { useT } from "@/src/shared/i18n";
import { kitMessages } from "../lib/messages";
import { cn } from "../lib/utils";
import { useFocusTrap } from "../hooks/useFocusTrap";
import { Button } from "./button";

/**
 * 앱 안 피드백 두 가지(Witems 화면 배치 규칙 §5):
 * - confirmAction: 위험 동작 확인 창(window.confirm 대신). 기본 포커스는 "취소", Esc·배경 = 취소.
 * - notify / notifyUndo: 화면 아래(넓은 화면은 오른쪽 위) 한 곳의 알림. 마우스를 올리거나 포커스가 있으면
 *   사라지지 않는다. 되돌릴 수 있는 동작은 확인 창 대신 실행 뒤 "되돌리기" 알림.
 * 둘 다 FeedbackHost(셸에 하나) 가 portalRoot()(Witems 는 ctx.portal) 에 그린다.
 */

export interface ConfirmOptions {
  title: string;
  description?: string;
  /** 설명 아래 짧은 목록(무엇이 바뀌고 무엇이 남는지) */
  details?: string[];
  confirmLabel: string;
  cancelLabel?: string;
  /** 빨간 확인 버튼(삭제·포기) */
  danger?: boolean;
}

export interface NoticeAction {
  label: string;
  run: () => void | Promise<unknown>;
}

export interface NoticeOptions {
  variant?: "info" | "success" | "error";
  action?: NoticeAction;
  /** ms, 기본 6초(동작이 있으면 8초) */
  duration?: number;
}

interface ConfirmEntry extends ConfirmOptions {
  id: number;
  resolve: (ok: boolean) => void;
}

interface NoticeEntry extends NoticeOptions {
  id: number;
  message: string;
}

interface FeedbackState {
  confirm: ConfirmEntry | null;
  notices: NoticeEntry[];
}

let state: FeedbackState = { confirm: null, notices: [] };
let seq = 0;
const listeners = new Set<() => void>();

function setState(next: FeedbackState) {
  state = next;
  for (const listener of listeners) { listener(); }
}

function subscribe(listener: () => void) {
  listeners.add(listener);
  return () => { listeners.delete(listener); };
}

const MAX_NOTICES = 3;

/** 앱 안 확인 창 — 확인이면 true. 이미 열린 창이 있으면 그 창은 취소로 닫는다. */
export function confirmAction(options: ConfirmOptions): Promise<boolean> {
  state.confirm?.resolve(false);
  return new Promise((resolve) => {
    seq += 1;
    setState({ ...state, confirm: { ...options, id: seq, resolve } });
  });
}

function closeConfirm(id: number, ok: boolean) {
  const current = state.confirm;
  if (!current || current.id !== id) { return; }
  setState({ ...state, confirm: null });
  current.resolve(ok);
}

/** 알림 하나를 띄우고 id 를 돌려준다. */
export function notify(message: string, options: NoticeOptions = {}): number {
  seq += 1;
  const entry: NoticeEntry = { ...options, id: seq, message };
  setState({ ...state, notices: [...state.notices, entry].slice(-MAX_NOTICES) });
  return entry.id;
}

export function dismissNotice(id: number) {
  if (!state.notices.some((n) => n.id === id)) { return; }
  setState({ ...state, notices: state.notices.filter((n) => n.id !== id) });
}

/** FeedbackHost 가 화면 언어의 문구("되돌리기")로 바꾸는 표지. */
const UNDO_LABEL = "\u0000undo";

/**
 * 실행이 끝난 동작의 "되돌리기" 알림. 누르면 undo 를 부르고 결과를 한 줄로 알린다.
 * 문구는 kit 기본값(화면 언어) — 호출하는 쪽이 locale 을 몰라도 되게 FeedbackHost 가 채운다.
 */
export function notifyUndo(message: string, undo: () => Promise<unknown>): number {
  return notify(message, { variant: "success", action: { label: UNDO_LABEL, run: undo } });
}

export function useFeedbackState(): FeedbackState {
  return React.useSyncExternalStore(subscribe, () => state, () => state);
}

/** 테스트용: 상태 비우기. */
export function resetFeedback() {
  state.confirm?.resolve(false);
  setState({ confirm: null, notices: [] });
}

function ConfirmDialog({ entry }: { entry: ConfirmEntry }) {
  const t = useT(kitMessages);
  const panelRef = React.useRef<HTMLDivElement>(null);
  const titleId = React.useId();
  const descId = React.useId();
  const cancel = React.useCallback(() => closeConfirm(entry.id, false), [entry.id]);
  useFocusTrap(panelRef, true, cancel);
  return (
    <div className="fixed inset-0 z-modal flex items-end justify-center p-4 pb-[calc(var(--safe-bottom)+16px)] sm:items-center">
      <div aria-hidden="true" className="absolute inset-0 bg-overlay animate-fade-in motion-reduce:animate-none" onClick={cancel} />
      <div
        ref={panelRef}
        role="alertdialog"
        aria-modal="true"
        aria-labelledby={titleId}
        aria-describedby={entry.description ? descId : undefined}
        className="relative w-full max-w-sm rounded-xl border border-line bg-canvas-elevated p-4 shadow-overlay animate-scale-in motion-reduce:animate-none"
      >
        <h2 id={titleId} className="text-base font-semibold text-fg">{entry.title}</h2>
        {entry.description ? <p id={descId} className="mt-1 text-sm text-fg-secondary">{entry.description}</p> : null}
        {entry.details && entry.details.length > 0 ? (
          <ul className="mt-2 list-disc space-y-1 break-keep pl-5 text-sm text-fg-secondary">
            {entry.details.map((line) => <li key={line}>{line}</li>)}
          </ul>
        ) : null}
        <div className="mt-4 flex justify-end gap-2">
          <Button variant="ghost" data-autofocus="" onClick={cancel}>{entry.cancelLabel ?? t.cancel}</Button>
          <Button variant={entry.danger ? "danger" : "primary"} onClick={() => closeConfirm(entry.id, true)}>{entry.confirmLabel}</Button>
        </div>
      </div>
    </div>
  );
}

function NoticeItem({ entry }: { entry: NoticeEntry }) {
  const t = useT(kitMessages);
  const [held, setHeld] = React.useState(false);
  const [running, setRunning] = React.useState(false);
  const duration = entry.duration ?? (entry.action ? 8000 : 6000);
  React.useEffect(() => {
    if (held || running || duration <= 0) { return undefined; }
    const timer = setTimeout(() => dismissNotice(entry.id), duration);
    return () => clearTimeout(timer);
  }, [held, running, duration, entry.id]);

  const action = entry.action;
  const label = action?.label === UNDO_LABEL ? t.undo : action?.label;
  async function run() {
    if (!action) { return; }
    setRunning(true);
    try {
      await action.run();
      dismissNotice(entry.id);
      if (action.label === UNDO_LABEL) { notify(t.undone); }
    } catch (err) {
      console.warn("punisher: notice action failed", err);
      dismissNotice(entry.id);
      notify(t.undoFailed, { variant: "error" });
    }
  }

  return (
    <div
      role={entry.variant === "error" ? "alert" : "status"}
      onMouseEnter={() => setHeld(true)}
      onMouseLeave={() => setHeld(false)}
      onFocus={() => setHeld(true)}
      onBlur={(event) => { if (!event.currentTarget.contains(event.relatedTarget as Node | null)) { setHeld(false); } }}
      className="pointer-events-auto flex w-full max-w-sm items-center gap-2 rounded-md border border-line bg-canvas-elevated py-1.5 pl-3 pr-1.5 text-sm text-fg shadow-overlay animate-fade-in motion-reduce:animate-none"
    >
      <span className={cn("min-w-0 flex-1 py-1", entry.variant === "error" && "text-danger")}>{entry.message}</span>
      {action && label ? (
        <Button variant="ghost" size="sm" loading={running} onClick={run} className="shrink-0 text-accent-text">{label}</Button>
      ) : null}
      <button
        type="button"
        aria-label={t.closeToast}
        onClick={() => dismissNotice(entry.id)}
        className="inline-flex size-8 shrink-0 items-center justify-center rounded-sm text-fg-muted hover:bg-canvas-hover hover:text-fg focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-focus"
      >
        <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" aria-hidden="true">
          <path d="M18 6 6 18M6 6l12 12" />
        </svg>
      </button>
    </div>
  );
}

/** 셸에 하나 둔다(AppShell·WitemsShell). */
export function FeedbackHost() {
  const { confirm, notices } = useFeedbackState();
  return createPortal(
    <>
      {confirm ? <ConfirmDialog key={confirm.id} entry={confirm} /> : null}
      <div
        aria-live="polite"
        className={cn(
          "pointer-events-none fixed inset-x-3 z-toast flex flex-col items-center gap-2",
          // 폰: 화면 아래(Witems 하단 탭 위), 넓은 화면: 오른쪽 위(셸 상단 바 아래)
          "bottom-[calc(var(--app-bottom-offset,var(--safe-bottom))+16px)]",
          "md:inset-x-auto md:bottom-auto md:right-4 md:top-[calc(var(--app-header-offset,0px)+16px)] md:items-end",
        )}
      >
        {notices.map((n) => <NoticeItem key={n.id} entry={n} />)}
      </div>
    </>,
    portalRoot(),
  );
}
