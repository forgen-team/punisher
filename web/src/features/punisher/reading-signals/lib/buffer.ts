import type { ReadingSignalInput } from "@/src/shared/api/punisher";

/** 블록 앵커("principles-3") — 섹션 앵커("principles")는 블록과 겹쳐 세지 않는다. */
export const BLOCK_ANCHOR = /-\d+$/;

/** 화면에 보이는 블록 중 체류 시간을 받을 하나 — 가장 많이 보이는 블록(최소 minShare). */
export function focusedAnchor(shares: Map<string, number>, minShare = 0.3): string | null {
  let best: string | null = null;
  let bestShare = minShare;
  for (const [anchor, share] of shares) {
    if (share >= bestShare) { best = anchor; bestShare = share; }
  }
  return best;
}

/** 보낼 때까지 모아 두는 블록별 체류 시간·용어 열기. */
export class SignalBuffer {
  private items = new Map<string, ReadingSignalInput>();

  private get(anchor: string): ReadingSignalInput {
    let item = this.items.get(anchor);
    if (!item) {
      item = { anchor, dwellMs: 0 };
      this.items.set(anchor, item);
    }
    return item;
  }

  dwell(anchor: string, ms: number): void {
    this.get(anchor).dwellMs += ms;
  }

  term(anchor: string, conceptId: string): void {
    const item = this.get(anchor);
    item.terms = { ...item.terms, [conceptId]: (item.terms?.[conceptId] ?? 0) + 1 };
  }

  /** 모은 것을 꺼내고 비운다(1초 미만 체류만 있는 블록은 버린다). */
  drain(): ReadingSignalInput[] {
    const out = [...this.items.values()].filter((i) => i.dwellMs >= 1000 || i.terms);
    this.items.clear();
    return out;
  }
}
