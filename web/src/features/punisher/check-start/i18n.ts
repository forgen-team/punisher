import { defineMessages } from "@/src/shared/i18n";

export const messages = defineMessages({
  ko: {
    knownToast: "이미 앎으로 표시했어요",
    check: "확인하기",
    notReviewed: "검수 후 확인할 수 있어요",
    verifyTitle: "짧게 확인할까요?",
    skip: "건너뛰기",
    undoFailed: "되돌리지 못했어요",
  },
  en: {
    knownToast: "Marked as known",
    check: "Check",
    notReviewed: "You can check once it's been reviewed",
    verifyTitle: "Do a quick check?",
    skip: "Skip",
    undoFailed: "Couldn't undo",
  },
});
