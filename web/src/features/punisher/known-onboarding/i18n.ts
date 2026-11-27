import { defineMessages } from "@/src/shared/i18n";

export const messages = defineMessages({
  ko: {
    sheetTitle: "이미 아는 개념 표시",
    areasLabel: "아는 영역",
    markKnown: "이미 앎으로 표시",
    saving: (done: number, total: number) => `저장 중 ${done}/${total}`,
    savedToast: (n: number) => `${n}개를 이미 앎으로 표시했어요`,
    saveFailed: (n: number) => `${n}개를 저장하지 못했어요. 남은 영역으로 다시 시도해 주세요.`,
    allMarked: "모두 표시됨",
  },
  en: {
    sheetTitle: "Mark concepts I know",
    areasLabel: "Areas I know",
    markKnown: "Mark as known",
    saving: (done: number, total: number) => `Saving ${done}/${total}`,
    savedToast: (n: number) => (n === 1 ? "Marked 1 concept as known" : `Marked ${n} concepts as known`),
    saveFailed: (n: number) =>
      n === 1 ? "Couldn't save 1 item. Try again with the remaining areas." : `Couldn't save ${n} items. Try again with the remaining areas.`,
    allMarked: "All marked",
  },
});
