import type { ExerciseVerdict } from "@/src/shared/api/punisher";
import { defineMessages, type Locale } from "@/src/shared/i18n";

export const messages = defineMessages({
  ko: {
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
    labTooLong: "앞 8KB만 읽어요",
    look: "보는 곳",
    expect: "예상",
  },
  en: {
    answerPlaceholder: "Your answer",
    submit: "Check answer",
    skip: "Skip and see solution",
    myAnswer: "Your answer",
    grading: "Writing feedback",
    gradeFailed: "Couldn't write feedback",
    labPlaceholder: "Remove passwords and tokens before pasting",
    labSubmit: "Read my result",
    labOutput: "Your output",
    labClear: "Clear",
    labTooLong: "Only the first 8 KB is read",
    look: "Where to look",
    expect: "Expected",
  },
});

export const VERDICT_LABELS: Record<Locale, Record<Exclude<ExerciseVerdict, "">, string>> = {
  ko: {
    correct: "맞아요",
    partial: "거의 맞아요",
    incorrect: "다시 볼 곳이 있어요",
    skipped: "건너뜀",
    matched: "예상과 같아요",
    different: "예상과 달라요",
    unreadable: "읽지 못했어요",
  },
  en: {
    correct: "Correct",
    partial: "Almost right",
    incorrect: "Some things to revisit",
    skipped: "Skipped",
    matched: "Matches the expected result",
    different: "Differs from the expected result",
    unreadable: "Couldn't read it",
  },
};
