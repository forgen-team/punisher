import { defineMessages } from "@/src/shared/i18n";

export const messages = defineMessages({
  ko: {
    addToReview: "복습에 넣기",
    inReview: "복습 카드 있음",
    sheetTitle: "복습 카드",
    sheetDescription: "내일부터 복습에 나와요. 질문과 답을 다듬어 저장하세요.",
    question: "질문",
    answer: "답",
    save: "저장",
    cancel: "취소",
    saved: "복습 카드를 만들었어요",
    loading: "불러오는 중",
    noAnswer: "아직 답이 없어요 — 답을 직접 적어 주세요.",
    undoFailed: "되돌리지 못했어요",
  },
  en: {
    addToReview: "Add to review",
    inReview: "In review",
    sheetTitle: "Review card",
    sheetDescription: "It shows up in reviews from tomorrow. Polish the question and answer, then save.",
    question: "Question",
    answer: "Answer",
    save: "Save",
    cancel: "Cancel",
    saved: "Review card created",
    loading: "Loading",
    noAnswer: "No answer yet — write it yourself.",
    undoFailed: "Couldn't undo",
  },
});
