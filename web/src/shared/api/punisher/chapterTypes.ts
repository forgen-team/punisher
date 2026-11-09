/** 다음 챕터 — 지도를 다 채운 뒤 고르거나 입력한 주제로 개념 묶음을 연다(punisher-server chapters). */

export type ChapterSuggestionKind = "proposed" | "deepen" | "trend" | "interest" | "new_area";

export interface ChapterSuggestion {
  id: string;
  kind: ChapterSuggestionKind;
  title: string;
  /** 왜 추천하는지 한 줄 */
  reason: string;
  score: number;
  conceptIds?: string[];
}

export interface ChapterSuggestionsResponse {
  question: string;
  suggestions: ChapterSuggestion[];
}

export type ChapterConceptAction = "create" | "reuse" | "promote";

export interface ChapterDraftConcept {
  id: string;
  key: string;
  action: ChapterConceptAction;
  name: string;
  area: string;
  depth: number;
  summary: string;
  objectives: { text: string; depth: number }[] | null;
  prerequisites: string[] | null;
}

export interface ChapterDraft {
  title: string;
  summary: string;
  /** 배우는 순서(선행 개념이 먼저) */
  concepts: ChapterDraftConcept[];
}

export interface ChapterDraftResponse {
  draftId: string;
  draft: ChapterDraft;
}

/** 오늘 화면의 '다음 챕터 고르기' 카드 근거 */
export interface NextChapterPrompt {
  remainingMinutes: number;
  thresholdMinutes: number;
  dailyMinutes: number;
}

export interface GraphChapter {
  id: string;
  title: string;
}
