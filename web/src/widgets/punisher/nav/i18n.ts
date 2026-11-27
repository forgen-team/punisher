import { defineMessages } from "@/src/shared/i18n";

export const messages = defineMessages({
  ko: { lectures: "강의", map: "지도", history: "기록", title: "학습", tabsLabel: "학습 보기", dueBadge: (n: number) => `복습 ${n}개` },
  en: { lectures: "Lectures", map: "Map", history: "History", title: "Learn", tabsLabel: "Learning views", dueBadge: (n: number) => `${n} reviews due` },
});
