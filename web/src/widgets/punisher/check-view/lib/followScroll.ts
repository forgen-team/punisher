/** 사용자가 답·채점·공개를 한 뒤 이 시간 안에 생긴 새 문항만 따라간다(튜터 턴 제한 3분과 같다). */
export const FOLLOW_WINDOW_MS = 3 * 60 * 1000;

export interface FollowInput {
  /** 마지막으로 입력창에서 동작한 시각(없으면 null) */
  actedAt: number | null;
  /** 마지막으로 사용자가 직접 스크롤한 시각(없으면 null) */
  scrolledAt: number | null;
  now: number;
}

/**
 * 새 현재 문항으로 데려갈지 — 바로 전에 사용자가 동작했고(FOLLOW_WINDOW_MS 안),
 * 그 뒤로 직접 스크롤하지 않았을 때만(위로 올려 읽는 중이면 끼어들지 않는다).
 */
export function shouldFollow({ actedAt, scrolledAt, now }: FollowInput, windowMs = FOLLOW_WINDOW_MS): boolean {
  if (actedAt === null || now - actedAt > windowMs) { return false; }
  return scrolledAt === null || scrolledAt < actedAt;
}

const SCROLL_KEYS = new Set(["ArrowUp", "ArrowDown", "PageUp", "PageDown", "Home", "End", " "]);

/** 키 입력이 사용자의 직접 스크롤인가 — 입력란 안의 키는 제외. */
export function isScrollKey(key: string, target: EventTarget | null): boolean {
  if (!SCROLL_KEYS.has(key)) { return false; }
  const el = target as HTMLElement | null;
  const tag = el?.tagName ?? "";
  return !(tag === "INPUT" || tag === "TEXTAREA" || tag === "SELECT" || el?.isContentEditable);
}

/** 문항 머리 요소 id(스크롤 대상). */
export function checkItemAnchorId(itemId: string): string {
  return `check-item-${itemId}`;
}
