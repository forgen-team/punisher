import { punisherFetch } from "./client";

/** W5a 학습 기능 API(서버 internal/http/handlers/study.go). */

export type MemoryModel = "ladder" | "fsrs";

export interface StudyPrefs {
  memoryModel: MemoryModel;
  /** 0.80 ~ 0.95 */
  targetRetention: number;
  /** 쉬는 요일 0=일 … 6=토 */
  restDays: number[];
  /** 복습 알림 시각(0~23), -1 = 끔 — 단독판 macOS 만 */
  reminderHour: number;
}

export type StudyPrefsPatch = Partial<StudyPrefs>;

export interface Leech {
  objectiveId: string;
  objectiveText: string;
  conceptId: string;
  conceptName: string;
  failCount: number;
  leechAt: string | null;
  suspendedAt: string | null;
}

export interface LeechesResponse {
  leeches: Leech[];
  suspended: Leech[];
  threshold: number;
}

export type LeechAction = "keep" | "suspend" | "resume";

export interface ForecastDay {
  day: string;
  objectives: number;
  cards: number;
}

export interface ForecastResponse {
  days: ForecastDay[];
  queued: number;
}

export interface ActivityDay {
  day: string;
  reviews: number;
  lectures: number;
  cards: number;
}

export interface ActivityResponse {
  days: ActivityDay[];
  today: { reviewsDone: number; reviewsDue: number; cardsDone: number; cardsDue: number; lecturesDone: number };
  restDays: number[];
  restToday: boolean;
  dailyMinutes: number;
}

export type PlanStatus = "no_deadline" | "done" | "on_track" | "tight" | "overdue";

export interface DeadlinePlan {
  dueOn: string;
  remainingConcepts: number;
  remainingMinutes: number;
  studyDays: number;
  minutesPerDay: number;
  dailyMinutes: number;
  finishOn: string;
  status: PlanStatus;
}

export type CardSourceKind = "note" | "thread";

export interface CardDraft {
  sourceKind: CardSourceKind;
  sourceId: string;
  lectureId: string;
  conceptId: string;
  anchor: string;
  question: string;
  answer: string;
  cardId: string;
}

export interface ReviewCard {
  id: string;
  /** external: 다른 앱(atlas 등)이 보낸 카드 — sourceUrl 은 출처 링크, stale 은 근거가 바뀜. */
  sourceKind: CardSourceKind | "external";
  sourceId: string;
  lectureId: string;
  conceptId: string;
  anchor: string;
  question: string;
  answer: string;
  reps: number;
  lapses: number;
  nextReviewAt: string | null;
  suspendedAt: string | null;
  createdAt: string;
  sourceUrl: string;
  externalId: string;
  stale: boolean;
}

/** 1 다시 · 2 어려움 · 3 알맞음 */
export type CardRating = 1 | 2 | 3;

export interface SearchHit {
  kind: "lecture" | "note" | "question" | "inquiry";
  refId: string;
  lectureId: string;
  anchor: string;
  title: string;
  /** 강의 결과의 섹션 제목(같은 강의가 여러 섹션에서 걸린다) — 그 밖은 "" */
  sectionTitle: string;
  /** 일치 부분은 [[ ]] 로 감싼다(3글자 이상 검색) */
  snippet: string;
}

const enc = encodeURIComponent;

export const studyApi = {
  getPrefs: () => punisherFetch<StudyPrefs>("/me/study-prefs"),
  savePrefs: (patch: StudyPrefsPatch) => punisherFetch<StudyPrefs>("/me/study-prefs", { method: "PUT", json: patch }),
  leeches: () => punisherFetch<LeechesResponse>("/me/leeches"),
  setLeech: (objectiveId: string, action: LeechAction) =>
    punisherFetch<{ objectiveId: string; action: LeechAction }>(`/objectives/${enc(objectiveId)}/leech`, { method: "POST", json: { action } }),
  forecast: (days = 14) => punisherFetch<ForecastResponse>(`/reviews/forecast?days=${days}`),
  activity: (days = 84) => punisherFetch<ActivityResponse>(`/me/activity?days=${days}`),
  goalPlan: (goalId: string) => punisherFetch<DeadlinePlan>(`/goals/${enc(goalId)}/plan`),
  deadlines: () => punisherFetch<{ deadlines: Record<string, string> }>("/goals/deadlines").then((r) => r.deadlines),
  setDeadline: (goalId: string, dueOn: string) =>
    punisherFetch<{ goalId: string; dueOn: string }>(`/goals/${enc(goalId)}/deadline`, { method: "PUT", json: { dueOn } }),
  cardDraft: (kind: CardSourceKind, sourceId: string) => punisherFetch<CardDraft>(`/cards/draft?kind=${kind}&sourceId=${enc(sourceId)}`),
  saveCard: (input: { kind: CardSourceKind; sourceId: string; question: string; answer: string }) =>
    punisherFetch<ReviewCard>("/cards", { method: "POST", json: input }),
  listCards: (due = false) => punisherFetch<{ cards: ReviewCard[] }>(`/cards${due ? "?due=1" : ""}`).then((r) => r.cards),
  gradeCard: (id: string, rating: CardRating) => punisherFetch<ReviewCard>(`/cards/${enc(id)}/grade`, { method: "POST", json: { rating } }),
  deleteCard: (id: string) => punisherFetch<void>(`/cards/${enc(id)}`, { method: "DELETE" }),
  search: (q: string, limit = 20) => punisherFetch<{ hits: SearchHit[] }>(`/search?q=${enc(q)}&limit=${limit}`).then((r) => r.hits),
};

export const studyKeys = {
  prefs: () => ["punisher", "study-prefs"] as const,
  leeches: () => ["punisher", "leeches"] as const,
  forecast: () => ["punisher", "forecast"] as const,
  activity: () => ["punisher", "activity"] as const,
  goalPlan: (goalId: string) => ["punisher", "goal-plan", goalId] as const,
  deadlines: () => ["punisher", "goal-deadlines"] as const,
  cardDraft: (kind: string, id: string) => ["punisher", "card-draft", kind, id] as const,
  cards: (due: boolean) => ["punisher", "cards", due ? "due" : "all"] as const,
  search: (q: string) => ["punisher", "search", q] as const,
};
