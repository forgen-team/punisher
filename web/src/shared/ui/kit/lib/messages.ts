import { defineMessages } from "@/src/shared/i18n";

/** 공용 UI 부품의 기본 문구(화면 언어를 따른다). */
export const kitMessages = defineMessages({
  ko: {
    close: "닫기",
    closeToast: "알림 닫기",
    loadFailed: "불러오지 못했어요.",
    retry: "다시 시도",
    more: "더 보기",
    loading: "불러오는 중",
    themeLight: "라이트",
    themeDark: "다크",
    themeSystem: "시스템",
  },
  en: {
    close: "Close",
    closeToast: "Dismiss notification",
    loadFailed: "Couldn't load this.",
    retry: "Try again",
    more: "More",
    loading: "Loading",
    themeLight: "Light",
    themeDark: "Dark",
    themeSystem: "System",
  },
});
