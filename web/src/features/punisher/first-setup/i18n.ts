import type { ChosenReaderLevel, Familiarity } from "@/src/shared/api/punisher";
import { defineMessages } from "@/src/shared/i18n";

export const messages = defineMessages({
  ko: {
    eyebrow: (step: number, total: number) => `처음 설정 · ${step} / ${total}`,
    levelTitle: "개발은 어느 정도 해 보셨어요?",
    levelHelper: "답에 맞춰 강의를 얼마나 풀어 쓸지 정해요. 시험 문제는 없고, 설정에서 언제든 바꿀 수 있어요.",
    levelGroup: "개발 경험",
    areasTitle: "영역마다 아는 만큼 골라 주세요",
    areasHelper: "‘설명 가능’을 고른 영역은 기본 개념을 이미 앎으로 두고 한 단계 깊은 곳부터 추천해요.",
    summaryTitle: "이렇게 시작할게요",
    summaryLevel: "설명 수준",
    summarySkipped: "건너뛰는 곳",
    skippedAreas: (areas: string) => `${areas} — 깊이 1 이미 앎`,
    skippedNone: "없음",
    summaryFoundations: "0단계 기초 트랙",
    foundationsStart: "여기서 시작",
    foundationsHidden: "숨김",
    note: "읽다가 ‘쉽게’를 자주 누르면 설명을 더 풀어 쓸지 한 번 물어볼게요. 수준은 설정 › 학습에서 바꿀 수 있어요.",
    skip: "건너뛰고 기본 강의로",
    next: "다음",
    back: "이전",
    start: "시작하기",
    savingSettings: "저장 중…",
    savingKnown: (done: number, total: number) => `이미 앎 저장 중 ${done}/${total}`,
    knownFailed: (n: number) => `${n}개를 이미 앎으로 저장하지 못했어요. 다시 시도하거나 나중에 지도 ⋯ 메뉴에서 표시해 주세요.`,
    retry: "다시 시도",
    close: "닫기",
  },
  en: {
    eyebrow: (step: number, total: number) => `First setup · ${step} / ${total}`,
    levelTitle: "How much development have you done?",
    levelHelper: "Your answer sets how much lectures spell things out. There's no test, and you can change it anytime in Settings.",
    levelGroup: "Development experience",
    areasTitle: "For each area, pick how well you know it",
    areasHelper: "Areas marked 'Can explain' treat the basic concepts as known and recommend from one level deeper.",
    summaryTitle: "Here's how you'll start",
    summaryLevel: "Explanation level",
    summarySkipped: "Skipped",
    skippedAreas: (areas: string) => `${areas} — depth 1 known`,
    skippedNone: "None",
    summaryFoundations: "Step 0 foundations track",
    foundationsStart: "Start here",
    foundationsHidden: "Hidden",
    note: "If you tap 'Simpler' often while reading, we'll ask once whether to spell things out more. You can change the level in Settings › Learning.",
    skip: "Skip, use standard lectures",
    next: "Next",
    back: "Back",
    start: "Get started",
    savingSettings: "Saving…",
    savingKnown: (done: number, total: number) => `Saving known ${done}/${total}`,
    knownFailed: (n: number) =>
      n === 1
        ? "Couldn't mark 1 concept as known. Try again, or mark it later from the map's ⋯ menu."
        : `Couldn't mark ${n} concepts as known. Try again, or mark them later from the map's ⋯ menu.`,
    retry: "Try again",
    close: "Close",
  },
});

/** 1단계 선택지 한 줄 설명(이름은 entities READER_LEVEL_LABELS_BY_LOCALE). */
export const LEVEL_DESCRIPTIONS = defineMessages<Record<ChosenReaderLevel, string>>({
  ko: {
    beginner: "HTTP, 터미널, 서버 같은 말이 낯설어요",
    novice: "학교·부트캠프·독학으로 뭔가 만들어 봤어요",
    practitioner: "API·DB는 다루지만 OS·네트워크 속사정은 약해요",
    advanced: "기본은 건너뛰고 내부 구현·장애 사례를 보고 싶어요",
  },
  en: {
    beginner: "Words like HTTP, terminal, and server are unfamiliar",
    novice: "I've built something at school, a bootcamp, or on my own",
    practitioner: "I work with APIs and databases, but OS and networking internals are fuzzy",
    advanced: "Skip the basics — show me internals and incident stories",
  },
});

export const FAMILIARITY_LABELS = defineMessages<Record<Familiarity, string>>({
  ko: { new: "처음", heard: "들어봄", can_explain: "설명 가능" },
  en: { new: "New", heard: "Heard of it", can_explain: "Can explain" },
});
