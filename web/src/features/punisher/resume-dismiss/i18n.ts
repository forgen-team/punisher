import { defineMessages } from "@/src/shared/i18n";

export const messages = defineMessages({
  ko: {
    percentRead: (percent: number) => `${percent}% 읽음`,
    menu: (title: string) => `${title} 동작`,
    dismiss: "이어보기에서 빼기",
    dismissed: "이어보기에서 뺐어요",
    undo: "되돌리기",
    undoFailed: "되돌리지 못했어요",
  },
  en: {
    percentRead: (percent: number) => `${percent}% read`,
    menu: (title: string) => `${title} actions`,
    dismiss: "Remove from Continue",
    dismissed: "Removed from Continue",
    undo: "Undo",
    undoFailed: "Couldn't undo",
  },
});
