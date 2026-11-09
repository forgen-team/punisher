import type { ChapterConceptAction, ChapterSuggestionKind } from "@/src/shared/api/punisher";

export const SUGGESTION_KIND_LABELS: Record<ChapterSuggestionKind, string> = {
  proposed: "자주 나온 개념",
  deepen: "더 깊게",
  trend: "요즘 뜨는 주제",
  interest: "내 관심 분야",
  "new_area": "새 영역",
};

export const CONCEPT_ACTION_LABELS: Record<ChapterConceptAction, string> = {
  create: "새 개념",
  promote: "제안해 둔 개념",
  reuse: "지도에 있는 개념",
};

export const FREE_TEXT_MAX = 500;
export const MAX_PICKS = 5;
