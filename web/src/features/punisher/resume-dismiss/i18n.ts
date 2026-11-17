import { defineMessages } from "@/src/shared/i18n";

export const messages = defineMessages({
  ko: {
    part: (index: number, total: number) => `${index}/${total}부분`,
    progress: (percent: number, minutes: number) => `${percent}% 읽음 · 약 ${minutes}분 남음`,
    menu: (title: string) => `${title} 강의 메뉴`,
    dismiss: "이어보기에서 제거",
    collapse: "접기",
    showAll: (n: number) => `모두 보기 (${n}개 더)`,
    dismissed: (label: string) => `‘${label}’을(를) 이어보기에서 뺐어요. 다시 열면 돌아와요.`,
    undo: "되돌리기",
    close: "닫기",
  },
  en: {
    part: (index: number, total: number) => `Part ${index}/${total}`,
    progress: (percent: number, minutes: number) => `${percent}% read · about ${minutes} min left`,
    menu: (title: string) => `${title} lecture menu`,
    dismiss: "Remove from Continue",
    collapse: "Show less",
    showAll: (n: number) => `Show all (${n} more)`,
    dismissed: (label: string) => `Removed ‘${label}’ from Continue. It comes back when you open it again.`,
    undo: "Undo",
    close: "Close",
  },
});
