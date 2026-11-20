import { defineMessages } from "@/src/shared/i18n";

export const messages = defineMessages({
  ko: {
    title: "설명을 더 풀어 쓸까요?",
    change: "바꾸기",
    dismiss: "괜찮아요",
    label: "설명 수준 제안",
    target: (level: string) => `바꿀 수준: ${level}`,
  },
  en: {
    title: "Want explanations spelled out more?",
    change: "Change",
    dismiss: "No thanks",
    label: "Explanation level suggestion",
    target: (level: string) => `New level: ${level}`,
  },
});
