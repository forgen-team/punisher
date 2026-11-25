import { defineMessages } from "@/src/shared/i18n";

export const messages = defineMessages({
  ko: {
    label: "학습 자료 검색",
    placeholder: "강의·메모·질문 해설에서 찾기",
    kind: (k: string) => ({ lecture: "강의", note: "메모", question: "강의 중 질문", inquiry: "질문 해설" })[k] ?? k,
    empty: (q: string) => `'${q}' 와 맞는 자료가 없어요.`,
    hint: "세 글자 이상이면 본문 전체에서, 더 짧으면 제목·본문에서 그대로 찾아요.",
    count: (n: number) => `${n}개`,
  },
  en: {
    label: "Search study material",
    placeholder: "Search lectures, notes and explanations",
    kind: (k: string) => ({ lecture: "Lecture", note: "Note", question: "Lecture question", inquiry: "Explanation" })[k] ?? k,
    empty: (q: string) => `Nothing matches '${q}'.`,
    hint: "Three or more characters search the full text; shorter queries match titles and text as typed.",
    count: (n: number) => `${n}`,
  },
});
