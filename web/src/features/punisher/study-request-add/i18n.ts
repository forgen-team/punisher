import { defineMessages } from "@/src/shared/i18n";

/** 단독판 전용 학습 요청 직접 추가 문구 */
export const messages = defineMessages({
  ko: {
    topicLabel: "공부하고 싶은 주제",
    placeholder: "예: 쿠버네티스 Pod 재시작",
    add: "추가",
  },
  en: {
    topicLabel: "Topic you want to study",
    placeholder: "e.g. Kubernetes Pod restarts",
    add: "Add",
  },
});
