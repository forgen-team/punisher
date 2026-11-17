import type { ChapterConceptAction, ChapterSuggestionKind } from "@/src/shared/api/punisher";
import { defineMessages } from "@/src/shared/i18n";

export const messages = defineMessages({
  ko: {
    defaultQuestion: "다음엔 어떤 걸 공부해 보고 싶어요?",
    suggestions: "추천",
    maxPicks: (n: number) => `최대 ${n}개까지 고를 수 있어요`,
    freeText: "직접 입력",
    charCount: (n: number, max: number) => `${n}/${max}자`,
    freeTextPlaceholder: "예) Kubernetes 네트워킹을 실무 수준으로",
    picked: (n: number) => `선택 ${n}개`,
    design: "챕터 설계해 보기",
    designing: "챕터를 설계하는 중… (1~3분)",
    // 제안 카드
    suggestionsEmptyTitle: "지금은 추천할 주제가 없어요",
    suggestionsEmptyDescription: "아래에 공부하고 싶은 걸 직접 적어 주세요.",
    // 미리보기
    depth: (n: number) => `깊이 ${n}`,
    objectiveCount: (n: number) => `목표 ${n}개`,
    previewLabel: "새 챕터 미리보기",
    newConcepts: (n: number) => `새로 열리는 개념 ${n}개 · 내 지도에만 보여요`,
    prerequisites: (names: string) => `먼저 알면 좋은 것: ${names}`,
    back: "다시 고르기",
    apply: "이대로 열기",
    // 배치 진단
    placementTitle: "배치 진단",
    placementStart: "시작",
    placementSkip: "건너뛰기",
    backToLearn: "학습",
  },
  en: {
    defaultQuestion: "What would you like to study next?",
    suggestions: "Suggestions",
    maxPicks: (n: number) => `Pick up to ${n}`,
    freeText: "Write your own",
    charCount: (n: number, max: number) => `${n}/${max} characters`,
    freeTextPlaceholder: "e.g. Kubernetes networking at a production level",
    picked: (n: number) => `${n} selected`,
    design: "Design a chapter",
    designing: "Designing the chapter… (1–3 min)",
    suggestionsEmptyTitle: "No topics to suggest right now",
    suggestionsEmptyDescription: "Write what you'd like to study below.",
    depth: (n: number) => `Depth ${n}`,
    objectiveCount: (n: number) => (n === 1 ? "1 objective" : `${n} objectives`),
    previewLabel: "New chapter preview",
    newConcepts: (n: number) => `${n === 1 ? "1 new concept" : `${n} new concepts`} · visible only on your map`,
    prerequisites: (names: string) => `Good to know first: ${names}`,
    back: "Pick again",
    apply: "Open as is",
    placementTitle: "Placement check",
    placementStart: "Start",
    placementSkip: "Skip",
    backToLearn: "Learn",
  },
});

export const SUGGESTION_KIND_LABELS = defineMessages<Record<ChapterSuggestionKind, string>>({
  ko: {
    proposed: "자주 나온 개념",
    deepen: "더 깊게",
    trend: "요즘 뜨는 주제",
    interest: "내 관심 분야",
    "new_area": "새 영역",
  },
  en: {
    proposed: "Frequently seen concept",
    deepen: "Go deeper",
    trend: "Trending topic",
    interest: "Your interests",
    "new_area": "New area",
  },
});

export const CONCEPT_ACTION_LABELS = defineMessages<Record<ChapterConceptAction, string>>({
  ko: { create: "새 개념", promote: "제안해 둔 개념", reuse: "지도에 있는 개념" },
  en: { create: "New concept", promote: "Proposed concept", reuse: "Already on the map" },
});
