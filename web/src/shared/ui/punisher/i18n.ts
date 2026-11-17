import { defineMessages } from "@/src/shared/i18n";

/** shared/ui/punisher 화면 문구 */
export const messages = defineMessages({
  ko: {
    deeperCount: (n: number) => `더 깊게 본 강의 ${n}개`,
    table: "표",
    close: "닫기",
    continuesBelow: "아래로 이어져요",
  },
  en: {
    deeperCount: (n: number) => (n === 1 ? "1 deeper lecture" : `${n} deeper lectures`),
    table: "Table",
    close: "Close",
    continuesBelow: "More below",
  },
});
