import { useCallback, useEffect, useRef, type RefObject } from "react";
import type { CheckSessionStatus } from "@/src/shared/api/punisher";
import { eventOrigin, getAppElementById } from "@/src/shared/lib/dom-root";
import { checkItemAnchorId, isScrollKey, shouldFollow } from "./followScroll";

interface FollowScrollInput {
  status: CheckSessionStatus | undefined;
  turnCount: number;
  /** 마지막 턴의 id·상태 — 채점 중 줄이 답으로 바뀌면(턴 수는 그대로) 다시 따라간다 */
  lastTurnKey: string;
  /** 지금 답할 문항(active 가 아니면 null) */
  currentId: string | null;
  endRef: RefObject<HTMLElement | null>;
  resultRef: RefObject<HTMLElement | null>;
}

/** 상단 바 아래 여백(문항의 scroll-mt-28 과 같다). */
const TOP_BAR_OFFSET = 112;

/**
 * 가장 최근 턴(끝 표지 바로 앞 목록의 마지막 항목)을 보인다. 끝 표지만 화면 아래로 맞추면 하단 고정 입력창이
 * 그 턴을 덮으므로, 턴의 아래 끝을 입력창 바로 위에 두고(바로 앞 내 답도 함께 보이게) 턴이 길면 머리를 상단 바
 * 아래에 둔다. 이미 다 보이면 움직이지 않는다.
 */
function revealLatestTurn(end: HTMLElement | null) {
  const turns = end?.previousElementSibling?.querySelectorAll("li");
  const latest = turns && turns.length > 0 ? turns[turns.length - 1] : null;
  if (!end || !latest) { return; }
  const composer = end.parentElement?.querySelector<HTMLElement>(".sticky");
  // 고정된 입력창의 위 끝(모바일 하단 내비 오프셋 포함) — 입력창이 없으면 화면 아래 끝.
  const bottomLimit = Math.min(window.innerHeight, composer?.getBoundingClientRect().top ?? window.innerHeight);
  const rect = latest.getBoundingClientRect();
  if (rect.top >= TOP_BAR_OFFSET && rect.bottom <= bottomLimit) { return; }
  const delta = Math.min(rect.bottom - bottomLimit, rect.top - TOP_BAR_OFFSET);
  const max = document.documentElement.scrollHeight - window.innerHeight;
  window.scrollTo({ top: Math.max(0, Math.min(window.scrollY + delta, max)), behavior: "smooth" });
}

/**
 * 확인 화면 스크롤 — 끝나면 결과 요약으로, 처음 열면 마지막 턴으로. 진행 중에는 바로 전에 입력창에서
 * 동작했고(돌려주는 markAct) 그 뒤로 직접 스크롤하지 않았을 때만 따라간다: 새 현재 문항이면 그 문항 머리로,
 * 같은 문항의 새 턴이면 마지막으로.
 */
export function useFollowScroll({ status, turnCount, lastTurnKey, currentId, endRef, resultRef }: FollowScrollInput): () => void {
  const actedAt = useRef<number | null>(null);
  const scrolledAt = useRef<number | null>(null);
  const seenItem = useRef<string | null>(null);
  const seenTurns = useRef(0);
  const markAct = useCallback(() => {
    actedAt.current = Date.now();
  }, []);

  useEffect(() => {
    const onScroll = () => {
      scrolledAt.current = Date.now();
    };
    const onKey = (event: KeyboardEvent) => {
      // Shadow DOM 안 입력창의 키는 window 에서 target 이 호스트로 보이므로 실제 요소로 판단한다.
      if (isScrollKey(event.key, eventOrigin(event))) { onScroll(); }
    };
    window.addEventListener("wheel", onScroll, { passive: true });
    window.addEventListener("touchmove", onScroll, { passive: true });
    window.addEventListener("keydown", onKey);
    return () => {
      window.removeEventListener("wheel", onScroll);
      window.removeEventListener("touchmove", onScroll);
      window.removeEventListener("keydown", onKey);
    };
  }, []);

  useEffect(() => {
    const itemChanged = currentId !== null && seenItem.current !== null && currentId !== seenItem.current;
    const firstTurns = seenTurns.current === 0;
    seenItem.current = currentId ?? seenItem.current;
    seenTurns.current = turnCount;
    if (status === "finished") {
      resultRef.current?.scrollIntoView({ behavior: "smooth", block: "start" });
      return;
    }
    if (status !== "active" || turnCount === 0) { return; }
    if (firstTurns) {
      revealLatestTurn(endRef.current);
      return;
    }
    if (!shouldFollow({ actedAt: actedAt.current, scrolledAt: scrolledAt.current, now: Date.now() })) { return; }
    if (itemChanged && currentId) {
      getAppElementById(checkItemAnchorId(currentId), endRef.current)?.scrollIntoView({ behavior: "smooth", block: "start" });
      return;
    }
    revealLatestTurn(endRef.current);
  }, [status, turnCount, lastTurnKey, currentId, endRef, resultRef]);

  return markAct;
}
