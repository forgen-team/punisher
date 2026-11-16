/** 구조 블록 라벨(설계 v2.2 §6) */
export const KEYPOINT_LABELS: Record<string, string> = {
  definition: "정의",
  caution: "주의",
  rule: "규칙",
};
export const KEYPOINT_FALLBACK_LABEL = "핵심";

export const BLOCK_LABELS = {
  pause: "예상해 보세요",
  showOutput: "출력 보기",
  exercise: "직접 해 보기",
  showAnswer: "풀이 보기",
  nextStep: "다음 단계",
  traceTable: "손계산 표",
  chart: "차트",
  showAll: "모두 보기",
  summary: "한 장 요약",
  diffBefore: "고치기 전",
  diffAfter: "고친 후",
  broken: "이 자료를 표시할 수 없어요",
  lab: "내 터미널에서 해 보기",
  labCommand: "명령",
  labLook: "보는 곳",
  labExpect: "예상",
} as const;
