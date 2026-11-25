import { defineMessages } from "@/src/shared/i18n";

/** 단독판 화면(pages) 문구 */
export const messages = defineMessages({
  ko: {
    notFoundTitle: "페이지를 찾을 수 없어요",
    backToToday: "오늘의 강의로 돌아가기",
    prompts: "강의 프롬프트",
    settings: "설정",
    sectionLearning: "학습",
    sectionAdvanced: "고급",
    promptsMeta: "강의를 만드는 지시문을 단계별로 보고 고쳐요",
    export: "데이터 내보내기",
    exportMeta: "강의·진도·기록을 파일 하나로 받아요",
    sectionScreen: "화면",
    sectionAi: "AI 연결",
    sectionAbout: "정보",
    exportFailed: "내보내지 못했어요. 잠시 뒤 다시 시도해 주세요.",
    exporting: "내보내는 중…",
  },
  en: {
    notFoundTitle: "Page not found",
    backToToday: "Back to today's lectures",
    prompts: "Lecture prompts",
    settings: "Settings",
    sectionLearning: "Learning",
    sectionAdvanced: "Advanced",
    promptsMeta: "See and edit the instructions that write lectures, stage by stage",
    export: "Export data",
    exportMeta: "Download lectures, progress and history as one file",
    sectionScreen: "Display",
    sectionAi: "AI connection",
    sectionAbout: "About",
    exportFailed: "Couldn't export. Try again in a moment.",
    exporting: "Exporting…",
  },
});
