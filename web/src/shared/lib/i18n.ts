import { defineMessages } from "@/src/shared/i18n";

/** 떠나기 확인(router.tsx useConfirmLeave) 문구 */
export const leaveMessages = defineMessages({
  ko: {
    unsavedTitle: "저장하지 않은 편집이 있어요",
    unsavedBody: "초안은 이 기기에 남아 있어 다시 열면 이어서 쓸 수 있어요.",
    unsavedLeave: "나가기",
  },
  en: {
    unsavedTitle: "You have unsaved edits",
    unsavedBody: "The draft stays on this device, so you can pick it up when you come back.",
    unsavedLeave: "Leave",
  },
});
