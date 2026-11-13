
import { useCallback, useEffect, useRef, type RefObject } from "react";
import type { CheckSessionStatus } from "@/src/shared/api/punisher";
import { checkItemAnchorId, isScrollKey, shouldFollow } from "./followScroll";

interface FollowScrollInput {
  status: CheckSessionStatus | undefined;
  turnCount: number;
  /** 지금 답할 문항(active 가 아니면 null) */
  currentId: string | null;
  endRef: RefObject<HTMLElement | null>;
  resultRef: RefObject<HTMLElement | null>;
}

/**
 * 확인 화면 스크롤 — 끝나면 결과 요약으로, 처음 열면 마지막 턴으로. 진행 중에는 바로 전에 입력창에서
 * 동작했고(돌려주는 markAct) 그 뒤로 직접 스크롤하지 않았을 때만 따라간다: 새 현재 문항이면 그 문항 머리로,
 * 같은 문항의 새 턴이면 마지막으로.
 */
export function useFollowScroll({ status, turnCount, currentId, endRef, resultRef }: FollowScrollInput): () => void {
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
      if (isScrollKey(event.key, event.target)) { onScroll(); }
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
      endRef.current?.scrollIntoView({ behavior: "smooth", block: "nearest" });
      return;
    }
    if (!shouldFollow({ actedAt: actedAt.current, scrolledAt: scrolledAt.current, now: Date.now() })) { return; }
    if (itemChanged && currentId) {
      document.getElementById(checkItemAnchorId(currentId))?.scrollIntoView({ behavior: "smooth", block: "start" });
      return;
    }
    endRef.current?.scrollIntoView({ behavior: "smooth", block: "nearest" });
  }, [status, turnCount, currentId, endRef, resultRef]);

  return markAct;
}
