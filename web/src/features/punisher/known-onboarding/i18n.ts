import { defineMessages } from "@/src/shared/i18n";

export const messages = defineMessages({
  ko: {
    promptRegion: "지도 시작 안내",
    promptText: "이미 아는 영역을 표시하면 지도가 열려요",
    promptStart: "표시하기",
    promptClose: "안내 닫기",
    later: "나중에",
    saving: (done: number, total: number) => `저장 중 ${done}/${total}`,
    saveCount: (n: number) => `${n}개 앎으로 저장`,
    sheetTitle: "이미 아는 영역 표시",
    sheetDescription: "아는 기본 개념을 체크하면 이어지는 개념의 잠금이 풀려요.",
    saveFailed: (n: number) => `${n}개를 저장하지 못했어요. 남은 항목으로 다시 시도해 주세요.`,
    clearAll: "전체 해제",
    markArea: "영역 전체 앎",
    allMarked: "모두 표시됨",
    alreadyMarked: "이미 표시됨",
  },
  en: {
    promptRegion: "Getting started with the map",
    promptText: "Mark the areas you already know to open up the map",
    promptStart: "Mark",
    promptClose: "Close tip",
    later: "Later",
    saving: (done: number, total: number) => `Saving ${done}/${total}`,
    saveCount: (n: number) => (n === 1 ? "Save 1 as known" : `Save ${n} as known`),
    sheetTitle: "Mark areas you know",
    sheetDescription: "Check the basic concepts you know to unlock the concepts that follow.",
    saveFailed: (n: number) =>
      n === 1 ? "Couldn't save 1 item. Try again with the remaining items." : `Couldn't save ${n} items. Try again with the remaining items.`,
    clearAll: "Clear all",
    markArea: "Mark whole area",
    allMarked: "All marked",
    alreadyMarked: "Already marked",
  },
});
