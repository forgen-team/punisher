import type { ExerciseVerdict } from "@/src/shared/api/punisher";

export const EXERCISE_LABELS = {
  answerPlaceholder: "내 답",
  submit: "답 확인",
  skip: "건너뛰고 풀이 보기",
  myAnswer: "내 답",
  grading: "평을 쓰는 중",
  gradeFailed: "평을 만들지 못했어요",
  labPlaceholder: "붙여 넣기 전에 비밀번호·토큰은 지우세요",
  labSubmit: "결과 읽어 보기",
  labOutput: "내 결과",
  labClear: "지우기",
  look: "보는 곳",
  expect: "예상",
} as const;

export const VERDICT_LABELS: Record<Exclude<ExerciseVerdict, "">, string> = {
  correct: "맞아요",
  partial: "거의 맞아요",
  incorrect: "다시 볼 곳이 있어요",
  skipped: "건너뜀",
  matched: "예상과 같아요",
  different: "예상과 달라요",
  unreadable: "읽지 못했어요",
};

export const VERDICT_TONE: Record<Exclude<ExerciseVerdict, "">, string> = {
  correct: "text-success",
  partial: "text-warning",
  incorrect: "text-danger",
  skipped: "text-fg-muted",
  matched: "text-success",
  different: "text-warning",
  unreadable: "text-fg-muted",
};
