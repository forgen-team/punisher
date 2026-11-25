/** 카드 복습 한 판의 진행: 지금 카드, 답을 보였는가, 본 장 수. 채점하면 다음 카드로. */
export interface CardSession {
  index: number;
  revealed: boolean;
  graded: number;
}

export const START: CardSession = { index: 0, revealed: false, graded: 0 };

export function reveal(s: CardSession): CardSession {
  return s.revealed ? s : { ...s, revealed: true };
}

/** 채점은 답을 본 뒤에만(먼저 떠올리고 확인하는 흐름). */
export function graded(s: CardSession): CardSession {
  if (!s.revealed) { return s; }
  return { index: s.index + 1, revealed: false, graded: s.graded + 1 };
}
