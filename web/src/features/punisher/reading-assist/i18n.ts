import { defineMessages } from "@/src/shared/i18n";

export const messages = defineMessages({
  ko: {
    explaining: "설명을 만드는 중",
    maxChars: (n: number) => `${n}자 이하로 골라 주세요`,
    tooLong: (n: number) => `너무 길어요(최대 ${n}자)`,
    explainThis: "이 부분 설명",
    section: (title: string) => `섹션: ${title}`,
    explainDescription: "고른 구절을 이 강의 맥락으로 풀어 드려요",
    retry: "다시 시도",
  },
  en: {
    explaining: "Writing explanation",
    maxChars: (n: number) => `Select ${n} characters or fewer`,
    tooLong: (n: number) => `Too long (max ${n} characters)`,
    explainThis: "Explain this",
    section: (title: string) => `Section: ${title}`,
    explainDescription: "Explains the selected passage in the context of this lecture",
    retry: "Try again",
  },
});
