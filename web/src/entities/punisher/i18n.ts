import { CHECK_MAX_ANSWER_CHARS, CHECK_MAX_DISPUTE_CHARS } from "@/src/shared/api/punisher";
import { defineMessages } from "@/src/shared/i18n";

/** entities/punisher 화면 문구(배지·레벨 점·추천 이유·확인/설정 오류). 상태 이름표(Record)는 ui/constants.ts 의 *_BY_LOCALE. */
export const messages = defineMessages({
  ko: {
    areaOther: "기타",
    studyRequestReason: "관심 있게 본 주제와 이어지는 개념이에요.",
    readThrough: "다 읽음",
    reviewUnreviewed: "검수 전",
    reviewNotPassed: "검수 패스를 통과하지 못했습니다",
    reviewIncomplete: "검수 미완료",
    instantLecture: "즉석 강의",
    masteryAria: (label: string, grader: string) => `숙달 ${label}${grader ? `, 채점 ${grader}` : ""}`,
    uncheckedObjectives: (n: number) => `새 목표 ${n}개 미확인`,
  },
  en: {
    areaOther: "Other",
    studyRequestReason: "Builds on a topic you showed interest in.",
    readThrough: "Read through",
    reviewUnreviewed: "Not reviewed",
    reviewNotPassed: "Did not pass the review pass",
    reviewIncomplete: "Review incomplete",
    instantLecture: "Instant lecture",
    masteryAria: (label: string, grader: string) => `Mastery ${label}${grader ? `, graded by ${grader}` : ""}`,
    uncheckedObjectives: (n: number) => (n === 1 ? "1 new objective unchecked" : `${n} new objectives unchecked`),
  },
});

/** 확인·복습 API 오류 code → 안내 */
export const checkErrorMessages = defineMessages({
  ko: {
    ANSWER_EMPTY: "답을 적어 주세요.",
    ANSWER_TOO_LONG: `답은 ${CHECK_MAX_ANSWER_CHARS}자까지 쓸 수 있어요.`,
    REASON_TOO_LONG: `사유는 ${CHECK_MAX_DISPUTE_CHARS}자까지 쓸 수 있어요.`,
    INVALID_VERDICT: "맞음·부분·틀림 중에서 골라 주세요.",
    NO_OBJECTIVES: "이 강의에는 확인할 학습 목표가 없어요.",
    INVALID_KIND: "지원하지 않는 확인 종류예요.",
    NO_REVIEWS: "오늘 복습할 항목이 없어요.",
    NO_PLACEMENT: "이 챕터에는 배치 진단할 개념이 부족해요.",
    MISSING_TARGET: "진단할 대상을 찾지 못했어요.",
    CHOICE_REQUIRED: "보기를 골라 주세요.",
    INVALID_CHOICE: "보기를 다시 골라 주세요.",
    TURN_REQUIRED: "인정할 오개념을 찾지 못했어요.",
    INVALID_STATUS: "지원하지 않는 오개념 목록이에요.",
    INVALID_CONFIDENCE: "확실·대충·모름 중에서 골라 주세요.",
    NO_SYNTHESIS: "종합 문제는 개념 3개 이상을 확인하면 일주일에 한 번 풀 수 있어요.",
    BAD_REQUEST: "요청을 처리하지 못했어요. 다시 시도해 주세요.",
  },
  en: {
    ANSWER_EMPTY: "Write an answer.",
    ANSWER_TOO_LONG: `Answers can be up to ${CHECK_MAX_ANSWER_CHARS} characters.`,
    REASON_TOO_LONG: `Reasons can be up to ${CHECK_MAX_DISPUTE_CHARS} characters.`,
    INVALID_VERDICT: "Choose correct, partial, or incorrect.",
    NO_OBJECTIVES: "This lecture has no learning objectives to check.",
    INVALID_KIND: "This check type isn't supported.",
    NO_REVIEWS: "Nothing to review today.",
    NO_PLACEMENT: "This chapter doesn't have enough concepts for a placement check.",
    MISSING_TARGET: "Couldn't find what to check.",
    CHOICE_REQUIRED: "Pick an option.",
    INVALID_CHOICE: "Pick an option again.",
    TURN_REQUIRED: "Couldn't find the misconception to acknowledge.",
    INVALID_STATUS: "This misconception list isn't supported.",
    INVALID_CONFIDENCE: "Choose sure, roughly, or don't know.",
    NO_SYNTHESIS: "Synthesis questions open once a week after you've checked at least 3 concepts.",
    BAD_REQUEST: "Couldn't process the request. Try again.",
  },
});

/** 설정 > 학습 PUT 400 code → 안내 */
export const settingsErrorMessages = defineMessages({
  ko: {
    INVALID_LANGUAGE: "지원하지 않는 예시 코드 언어예요.",
    INVALID_GRADER: "자기 채점·튜터 채점 중에서 골라 주세요.",
    INVALID_DAILY_MINUTES: "하루 학습량은 15·30·45·60분 중에서 골라 주세요.",
    INVALID_MASTERY_TARGET: "숙달 기준은 레벨 3~5 중에서 골라 주세요.",
    INVALID_INTERVALS: "복습 간격은 점점 커지는 1 이상의 정수(일) 1~6개로 적어 주세요.",
    INVALID_REVIEW_MINUTES: "하루 복습 시간은 0~30분 중에서 골라 주세요.",
    INVALID_LAB_OS: "macOS·Linux·쓰지 않음 중에서 골라 주세요.",
    INVALID_READER_LEVEL: "설명 수준을 네 가지 중에서 골라 주세요.",
    INVALID_AREA_FAMILIARITY: "영역마다 처음·들어봄·설명 가능 중에서 골라 주세요.",
    INVALID_AI_PROVIDER: "Claude Code·Codex·Gemini(Antigravity) 중에서 골라 주세요.",
    SETTINGS_REQUIRED: "바꿀 설정을 골라 주세요.",
    BAD_REQUEST: "설정을 저장하지 못했어요. 다시 시도해 주세요.",
  },
  en: {
    INVALID_LANGUAGE: "This example code language isn't supported.",
    INVALID_GRADER: "Choose self-grading or tutor grading.",
    INVALID_DAILY_MINUTES: "Choose a daily study time of 15, 30, 45, or 60 minutes.",
    INVALID_MASTERY_TARGET: "Choose a mastery target between level 3 and 5.",
    INVALID_INTERVALS: "Enter 1 to 6 review intervals in days, each a whole number of at least 1 and increasing.",
    INVALID_REVIEW_MINUTES: "Choose a daily review time between 0 and 30 minutes.",
    INVALID_LAB_OS: "Choose macOS, Linux, or none.",
    INVALID_READER_LEVEL: "Choose one of the four explanation levels.",
    INVALID_AREA_FAMILIARITY: "For each area, choose new, heard of it, or can explain.",
    INVALID_AI_PROVIDER: "Choose Claude Code, Codex, or Gemini (Antigravity).",
    SETTINGS_REQUIRED: "Choose a setting to change.",
    BAD_REQUEST: "Couldn't save settings. Try again.",
  },
});
