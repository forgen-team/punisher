import type { Question } from "@/src/shared/api/punisher";

/** 루트 질문 + 이어 묻기(오래된 순). */
export interface ReadingThread {
  root: Question;
  replies: Question[];
  /** 마지막 대화 시각 — 최신 스레드를 위로 */
  lastAt: string;
}

/** 같은 구절(앵커 + 인용)에 달린 스레드들(최신 순). */
export interface PassageThreads {
  key: string;
  anchor: string;
  quote: string;
  threads: ReadingThread[];
  /** 본문 '대화 n' — 이 구절의 살아 있는(답했거나 답 쓰는 중인 차례가 있는) 루트 스레드 수. 이어 묻기·실패만 남은 스레드는 세지 않는다 */
  exchanges: number;
}

const byCreated = (a: Question, b: Question) => a.createdAt.localeCompare(b.createdAt);

/** 서버의 평평한 질문 목록 → 스레드(parentId 로 묶는다, 루트는 parentId ""). 이어 묻기는 quote 가 "" 이다. 루트가 목록에 없는 이어 묻기는 그 자체를 루트로 둔다. */
export function groupThreads(questions: Question[]): ReadingThread[] {
  const ids = new Set(questions.map((q) => q.id));
  const replies = new Map<string, Question[]>();
  const roots: Question[] = [];
  for (const q of questions) {
    const parent = q.parentId ?? "";
    if (parent && ids.has(parent)) {
      replies.set(parent, [...(replies.get(parent) ?? []), q]);
    } else {
      roots.push(q);
    }
  }
  return roots
    .map((root) => {
      const list = (replies.get(root.id) ?? []).sort(byCreated);
      const last = list[list.length - 1] ?? root;
      return { root, replies: list, lastAt: last.createdAt };
    })
    .sort((a, b) => b.lastAt.localeCompare(a.lastAt));
}

export function passageKey(anchor: string, quote: string): string {
  return `${anchor}\u0000${quote}`;
}

/** 답했거나 답 쓰는 중인 차례가 하나라도 있는 스레드 — 실패만 남은 루트(다시 시도 전)는 개수에서 뺀다. */
export function isLiveThread(thread: ReadingThread): boolean {
  return thread.root.status !== "failed" || thread.replies.some((r) => r.status !== "failed");
}

/** 구절 스레드만(인용이 있는 루트) 구절별로 묶는다. 순서는 최신 대화 순. */
export function groupPassages(threads: ReadingThread[]): PassageThreads[] {
  const passages = new Map<string, PassageThreads>();
  for (const thread of threads) {
    const quote = thread.root.quote ?? "";
    if (!quote) { continue; }
    const key = passageKey(thread.root.anchor, quote);
    const passage = passages.get(key) ?? { key, anchor: thread.root.anchor, quote, threads: [], exchanges: 0 };
    passage.threads.push(thread);
    if (isLiveThread(thread)) { passage.exchanges += 1; }
    passages.set(key, passage);
  }
  return [...passages.values()];
}

/** 루트 id 로 스레드 찾기. 이어 묻기 id 를 주면 그 루트 스레드. */
export function findThread(threads: ReadingThread[], id: string): ReadingThread | undefined {
  return threads.find((t) => t.root.id === id || t.replies.some((r) => r.id === id));
}

/** 강의 툴바 배지 — 구절 스레드 루트 수(살아 있는 것만). */
export function countPassageThreads(passages: PassageThreads[]): number {
  return passages.reduce((sum, p) => sum + p.exchanges, 0);
}

/** 구절별 목록에 보이는 질문 id(루트 + 이어 묻기) — 나머지(예전 방식 질문·루트가 없는 이어 묻기)는 예전 목록에 남긴다. */
export function passageQuestionIds(passages: PassageThreads[]): Set<string> {
  const ids = new Set<string>();
  for (const passage of passages) {
    for (const thread of passage.threads) {
      ids.add(thread.root.id);
      for (const reply of thread.replies) { ids.add(reply.id); }
    }
  }
  return ids;
}
