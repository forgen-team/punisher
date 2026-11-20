import type { LabOS } from "@/src/shared/api/punisher";
import { defineMessages } from "@/src/shared/i18n";

export const messages = defineMessages({
  ko: {
    readerLevel: "설명 수준",
    readerLevelHint: "새로 만드는 강의·설명을 얼마나 풀어 쓸지 정해요. 가르치는 내용은 같아요. 이미 만든 강의는 바뀌지 않아요.",
    readerLevelUnset: "아직 고르지 않음",
    exampleLanguage: "예시 코드 언어",
    exampleLanguageHint: "강의·질문 답변·'이 부분 설명'의 코드 예시를 이 언어로 써요. 셸 명령·SQL·설정 파일은 원래 형식 그대로예요. 이미 만든 강의는 바뀌지 않아요.",
    dailyMinutes: "하루 학습량",
    dailyMinutesHint: "남은 강의가 이 속도로 2주 치보다 적어지면 오늘 화면에서 '다음 챕터 고르기'를 띄워요.",
    dailyOption: (m: number) => `하루 ${m}분`,
    labOs: "실습 환경",
    pretest: "강의 전 예측 질문",
    saving: "저장 중…",
    saved: "저장했어요.",
    // 복습
    masteryTarget: "숙달 기준",
    masteryOption: (level: number, label: string) => `레벨 ${level} · ${label}`,
    reviewIntervals: "복습 간격(일)",
    reviewMinutes: "하루 복습 시간",
    minutes: (m: number) => `${m}분`,
    adaptiveIntervals: "복습 간격 자동 조절",
    reviewGrader: "복습 채점 방식",
    // 간격 검증
    intervalsCount: (max: number) => `간격은 1~${max}개로 적어 주세요.`,
    intervalsInteger: "간격은 1 이상의 정수(일)로 적어 주세요.",
    intervalsIncreasing: "간격은 점점 커져야 해요.",
  },
  en: {
    readerLevel: "Explanation level",
    readerLevelHint: "How much new lectures and explanations spell things out. What they teach stays the same. Existing lectures don't change.",
    readerLevelUnset: "Not chosen yet",
    exampleLanguage: "Example code language",
    exampleLanguageHint: "Code examples in lectures, question answers, and 'Explain this' use this language. Shell commands, SQL, and config files keep their own format. Existing lectures don't change.",
    dailyMinutes: "Daily study time",
    dailyMinutesHint: "When remaining lectures drop below two weeks at this pace, Today suggests picking the next chapter.",
    dailyOption: (m: number) => `${m} min a day`,
    labOs: "Lab environment",
    pretest: "Prediction questions before lectures",
    saving: "Saving…",
    saved: "Saved.",
    masteryTarget: "Mastery target",
    masteryOption: (level: number, label: string) => `Level ${level} · ${label}`,
    reviewIntervals: "Review intervals (days)",
    reviewMinutes: "Daily review time",
    minutes: (m: number) => `${m} min`,
    adaptiveIntervals: "Adjust review intervals automatically",
    reviewGrader: "Review grading",
    intervalsCount: (max: number) => `Enter 1 to ${max} intervals.`,
    intervalsInteger: "Intervals must be whole numbers of days, 1 or more.",
    intervalsIncreasing: "Each interval must be longer than the last.",
  },
});

export const LAB_OS_LABELS = defineMessages<Record<LabOS, string>>({
  ko: { macos: "macOS", linux: "Linux", none: "쓰지 않음" },
  en: { macos: "macOS", linux: "Linux", none: "None" },
});

/** 언어 선택 — 어느 언어로 보고 있어도 찾을 수 있게 라벨·선택지는 두 언어에서 같다. */
export const LANGUAGE_LABEL = "언어 / Language";

/** 언어 이름은 그 언어로(어느 화면 언어에서도 같다). */
export const LOCALE_NAMES = { ko: "한국어", en: "English" } as const;
