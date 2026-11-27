import type { CheckSessionStatus, Confidence } from "@/src/shared/api/punisher";
import { defineMessages, type Locale } from "@/src/shared/i18n";

/** 목적격 조사(을/를) — 마지막 글자 받침으로 가른다. 한글이 아니면 '를'. */
export function objectParticle(word: string): string {
  const last = word.trim().slice(-1).charCodeAt(0);
  const isHangul = last >= 0xac00 && last <= 0xd7a3;
  return isHangul && (last - 0xac00) % 28 !== 0 ? "을" : "를";
}

export const messages = defineMessages({
  ko: {
    viewInLecture: "강의에서 보기",
    reviewTitle: "복습",
    checkTitle: "확인하기",
    checkOf: (lecture: string) => `확인하기 · ${lecture}`,
    skip: "건너뛰기",
    menuStop: "그만두기",
    stopConfirmTitle: (name: string) => `'${name}'${objectParticle(name)} 여기서 그만둘까요?`,
    stopConfirmReview: (answered: number, remaining: number) => `답한 ${answered}개는 저장하고, 남은 ${remaining}개는 내일 복습으로 넘겨요.`,
    stopConfirmOther: (answered: number, remaining: number) => `답한 ${answered}개는 저장하고, 남은 ${remaining}개는 하지 않고 끝내요.`,
    skippedToast: "건너뛰었어요",
    cancel: "취소",
    cancelledToast: "문제 만들기를 취소했어요",
    cancelFailed: "취소하지 못했어요. 다시 시도해 주세요",
    cancelTooLate: "이미 끝난 작업이라 취소할 수 없어요",
    undoFailed: "되돌리지 못했어요",
    noItems: "확인할 문항이 없어요",
    itemProgress: (index: number, total: number) => `문항 ${index}/${total}`,
    selfGrade: "자기 채점",
    answer: "답",
    reveal: "그냥 알려줘",
    send: "보내기",
    nextItem: "다음 문항",
    seeResult: "결과 보기",
    progress: "진행",
    finishedReview: (n: number) => `복습 ${n}개를 마쳤어요`,
    finishedItems: (n: number) => `문항 ${n}개를 마쳤어요`,
    continueUnchecked: (n: number) => `확인 대기 ${n}개 이어서`,
    confidence: "확신도",
    dispute: "내 답이 맞아요",
    myAnswer: "내 답: ",
    tutor: "튜터: ",
    turnFailed: "답을 만들지 못했어요.",
    retry: "다시 시도",
    missing: "놓친 기준",
    missingLine: (list: string) => `놓친 기준: ${list}`,
    modelAnswer: "모범답",
    criteria: "기준",
    revisit: "다시 볼 곳",
    revisitLine: (label: string) => `다시 볼 곳: ${label}`,
    result: "결과",
    markComplete: "강의 완료로 표시",
    today: "오늘로",
    queued: "복습이 밀려 대기",
    nextReview: (date: string) => `다음 복습 ${date}`,
    relearn: "다시 배우기",
    relearned: "다시 배우기로 바꿈",
    predictionBefore: "처음 예측",
    predictionNow: "지금 답",
    confidentErrors: (n: number) => `착각 ${n}`,
    placementCandidates: "이미 앎 후보",
    placementNone: "이미 앎 후보가 없어요",
    placementConfirm: "이미 앎으로 표시",
    placementConfirmed: "이미 앎으로 표시함",
    placementMap: "지도에서 보기",
    placementFailed: (n: number) => `${n}개를 표시하지 못했어요. 다시 눌러 주세요.`,
    misconceptionRevealed: (text: string) => `드러난 오개념: ${text}`,
    misconceptionAcknowledge: "맞아요, 그렇게 알았어요",
    misconceptionAcknowledged: "인정함",
    revealedNotice: "답을 공개했어요",
    skippedNotice: "건너뛰었어요",
    skippedRow: "건너뜀",
    tally: (correct: number, again: number, skipped: number) =>
      [`맞힘 ${correct}`, `다시 볼 것 ${again}`, ...(skipped > 0 ? [`건너뜀 ${skipped}`] : [])].join(" · "),
    lectureSummary: "이 강의 한 장 요약",
  },
  en: {
    viewInLecture: "View in lecture",
    reviewTitle: "Review",
    checkTitle: "Check",
    checkOf: (lecture: string) => `Check · ${lecture}`,
    skip: "Skip",
    menuStop: "Stop",
    stopConfirmTitle: (name: string) => `Stop '${name}' here?`,
    stopConfirmReview: (answered: number, remaining: number) => `Your ${answered} answers are saved, and the other ${remaining} move to tomorrow's review.`,
    stopConfirmOther: (answered: number, remaining: number) => `Your ${answered} answers are saved, and the other ${remaining} are left undone.`,
    skippedToast: "Skipped",
    cancel: "Cancel",
    cancelledToast: "Question creation cancelled",
    cancelFailed: "Couldn't cancel. Try again",
    cancelTooLate: "It already finished, so it can't be cancelled",
    undoFailed: "Couldn't undo",
    noItems: "No questions to check",
    itemProgress: (index: number, total: number) => `Question ${index}/${total}`,
    selfGrade: "Self-grade",
    answer: "Answer",
    reveal: "Just tell me",
    send: "Send",
    nextItem: "Next question",
    seeResult: "See results",
    progress: "Progress",
    finishedReview: (n: number) => (n === 1 ? "Finished 1 review" : `Finished ${n} reviews`),
    finishedItems: (n: number) => (n === 1 ? "Finished 1 question" : `Finished ${n} questions`),
    continueUnchecked: (n: number) => `Continue with ${n} waiting to check`,
    confidence: "Confidence",
    dispute: "My answer is right",
    myAnswer: "My answer: ",
    tutor: "Tutor: ",
    turnFailed: "Couldn't write a reply.",
    retry: "Try again",
    missing: "Missed criteria",
    missingLine: (list: string) => `Missed criteria: ${list}`,
    modelAnswer: "Model answer",
    criteria: "Criteria",
    revisit: "Revisit",
    revisitLine: (label: string) => `Revisit: ${label}`,
    result: "Result",
    markComplete: "Mark lecture complete",
    today: "Back to today",
    queued: "Waiting — reviews are backed up",
    nextReview: (date: string) => `Next review ${date}`,
    relearn: "Relearn",
    relearned: "Set to relearn",
    predictionBefore: "First prediction",
    predictionNow: "Answer now",
    confidentErrors: (n: number) => `Confidently wrong ${n}`,
    placementCandidates: "Possibly known",
    placementNone: "Nothing looks already known",
    placementConfirm: "Mark as known",
    placementConfirmed: "Marked as known",
    placementMap: "View on map",
    placementFailed: (n: number) => (n === 1 ? "Couldn't mark 1 concept. Please tap again." : `Couldn't mark ${n} concepts. Please tap again.`),
    misconceptionRevealed: (text: string) => `Misconception found: ${text}`,
    misconceptionAcknowledge: "Yes, that's what I thought",
    misconceptionAcknowledged: "Acknowledged",
    revealedNotice: "Answer revealed",
    skippedNotice: "Skipped",
    skippedRow: "Skipped",
    tally: (correct: number, again: number, skipped: number) =>
      [`Correct ${correct}`, `To revisit ${again}`, ...(skipped > 0 ? [`Skipped ${skipped}`] : [])].join(" · "),
    lectureSummary: "One-page lecture summary",
  },
});

/** 확신도(v3 §3) */
export const CONFIDENCE_ORDER: readonly Confidence[] = ["sure", "unsure", "unknown"];
export const CONFIDENCE_LABELS: Record<Locale, Record<Confidence, string>> = {
  ko: { sure: "확실", unsure: "대충", unknown: "모름" },
  en: { sure: "Sure", unsure: "Not sure", unknown: "No idea" },
};

/** 이의 제기 재채점 결과(conflict: 받아들였지만 그 뒤 다른 근거가 있어 상자는 되돌리지 않음) */
export const DISPUTE_RESULT_LABELS: Record<Locale, Record<"accepted" | "rejected" | "conflict", string>> = {
  ko: { accepted: "이의 받아들임", rejected: "원래 판정 유지", conflict: "이후 기록이 있어 상자 유지" },
  en: { accepted: "Dispute accepted", rejected: "Original verdict kept", conflict: "Box kept due to later records" },
};

/** 실행 중 표시(튜터 턴 · 이의 재채점) */
export const PENDING_LABELS: Record<Locale, Record<"turn" | "dispute", string>> = {
  ko: { turn: "튜터가 보는 중이에요", dispute: "다시 채점하는 중이에요" },
  en: { turn: "The tutor is looking", dispute: "Regrading" },
};

/** 세션 상태 줄(active 는 표시 없음) */
export const SESSION_STATUS_LABELS: Record<Locale, Record<CheckSessionStatus, string>> = {
  ko: {
    preparing: "문항을 만드는 중이에요",
    active: "",
    finished: "확인을 마쳤어요",
    abandoned: "포기한 확인이에요",
    failed: "문항을 만들지 못했어요",
  },
  en: {
    preparing: "Creating questions",
    active: "",
    finished: "Check finished",
    abandoned: "You gave up this check",
    failed: "Couldn't create questions",
  },
};

/** 복습 세션 상태 줄(active 는 표시 없음) */
export const REVIEW_STATUS_LABELS: Record<Locale, Record<CheckSessionStatus, string>> = {
  ko: {
    preparing: "복습 문항을 준비하는 중이에요",
    active: "",
    finished: "복습을 마쳤어요",
    abandoned: "그만둔 복습이에요",
    failed: "복습 문항을 준비하지 못했어요",
  },
  en: {
    preparing: "Preparing review questions",
    active: "",
    finished: "Review finished",
    abandoned: "You stopped this review",
    failed: "Couldn't prepare review questions",
  },
};

type DiagnosticKind = "pretest" | "known_verify" | "placement" | "synthesis";

/** 상단 제목(확인·복습 외 진단 세션) */
export const DIAGNOSTIC_TITLES: Record<Locale, Record<DiagnosticKind, string>> = {
  ko: { pretest: "예측 질문", "known_verify": "짧은 확인", placement: "배치 확인", synthesis: "종합 문제" },
  en: { pretest: "Prediction questions", "known_verify": "Quick check", placement: "Placement check", synthesis: "Synthesis" },
};

/** 종합 문제 결과(v3 §5) — 상자는 움직이지 않는다. */
export const SYNTHESIS_RESULT_LABELS: Record<Locale, Record<"passed" | "pulled" | "skipped" | "recorded", string>> = {
  ko: { passed: "통과 — 함께 설명할 수 있어요", pulled: "관련 복습을 앞당겼어요", skipped: "건너뜀 — 관련 복습을 앞당겼어요", recorded: "기록만 남겼어요" },
  en: { passed: "Passed — you can explain them together", pulled: "Related reviews moved up", skipped: "Skipped — related reviews moved up", recorded: "Recorded only" },
};
