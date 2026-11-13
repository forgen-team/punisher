import { punisherFetch } from "./client";
import type {
  CompleteResult,
  Concept,
  ConceptLecture,
  ConceptDetail,
  FeedbackKind,
  FeedbackResponse,
  FrontierItem,
  GenerationJob,
  Goal,
  GoalInput,
  GoalPathResponse,
  Graph,
  InstantRequest,
  InstantResponse,
  Lecture,
  MyLecture,
  MyLectureStatus,
  MyQuestion,
  Note,
  NoteInput,
  Page,
  Question,
  StudyRequest,
  TodayResponse,
  UserConceptState,
} from "./types";
import type { LearningSettingsPatch, LearningSettingsResponse, LectureLinks, SelectionExplanation } from "./learningTypes";
import type { PromptHistoryEntry, PromptPreview, PromptSaveResponse, PromptsResponse } from "./promptTypes";
import type { ChapterDraftResponse, ChapterSuggestionsResponse } from "./chapterTypes";
import type { CompletedArea, CompletedItem, Inquiry, InquiryFollowup } from "./inquiryTypes";
import type { CheckDetail, ConceptMastery, DiagnosticChoice, DiagnosticRequest } from "./checkTypes";
import type { CheckHistoryKind, CheckHistoryPage, ConceptMasterySummary, MasteryMetrics, ReviewSummary } from "./reviewTypes";
import type { Misconception, MisconceptionStatus } from "./misconceptionTypes";

const enc = encodeURIComponent;

function query(params: Record<string, string | number | undefined>): string {
  const search = new URLSearchParams();
  for (const [key, value] of Object.entries(params)) {
    if (value !== undefined && value !== "") { search.set(key, String(value)); }
  }
  const text = search.toString();
  return text ? `?${text}` : "";
}

export const punisherApi = {
  // Concepts & graph
  listConcepts: (area?: string) =>
    punisherFetch<{ concepts: Concept[] }>(`/concepts${query({ area })}`).then((r) => r.concepts),
  getConcept: (id: string) => punisherFetch<ConceptDetail>(`/concepts/${enc(id)}`),
  listConceptLectures: (id: string) =>
    punisherFetch<{ lectures: ConceptLecture[] }>(`/concepts/${enc(id)}/lectures`).then((r) => r.lectures),
  getGraph: (area?: string) => punisherFetch<Graph>(`/graph${query({ area })}`),
  getFrontier: (limit = 10) =>
    punisherFetch<{ items: FrontierItem[]; total: number }>(`/me/frontier${query({ limit })}`),
  setConceptState: (id: string, state: UserConceptState) =>
    punisherFetch<{ conceptId: string; state: string }>(`/me/concepts/${enc(id)}/state`, {
      method: "PUT",
      json: { state },
    }),
  /** queueVerify: 일괄 '이미 앎' — 목표를 진단 대기열에 넣는다(복습에 하루 1~2개씩 섞임). */
  markKnown: (id: string, options: { queueVerify?: boolean } = {}) =>
    punisherFetch<{ conceptId: string; state: string }>(`/concepts/${enc(id)}/known`, {
      method: "POST",
      json: options.queueVerify ? { queueVerify: true } : undefined,
    }),
  getGloss: (conceptId: string) =>
    punisherFetch<{ conceptId: string; gloss: string; cached: boolean }>(`/terms/${enc(conceptId)}/gloss`),

  // Today & study requests
  getToday: (limit = 5) => punisherFetch<TodayResponse>(`/today${query({ limit })}`),
  listStudyRequests: (limit = 20) =>
    punisherFetch<{ studyRequests: StudyRequest[] }>(`/study-requests${query({ limit })}`).then(
      (r) => r.studyRequests,
    ),
  /** 단독판: 직접 입력한 학습 요청 추가 (source=manual). 이미 있는 문구면 기존 요청을 돌려준다. */
  createStudyRequest: (query: string) =>
    punisherFetch<{ studyRequest: StudyRequest; created: boolean }>("/study-requests", {
      method: "POST",
      json: { source: "manual", query },
    }),
  /** 개념 찾기 다시 시도 — 즉시 백그라운드로 시작(202) */
  dismissStudyRequest: (id: string) =>
    punisherFetch<{ studyRequestId: string; dismissed: boolean }>(`/study-requests/${enc(id)}/dismiss`, { method: "POST" }),
  restoreStudyRequest: (id: string) =>
    punisherFetch<{ studyRequestId: string; dismissed: boolean }>(`/study-requests/${enc(id)}/dismiss`, { method: "DELETE" }),
  resolveStudyRequest: (id: string) =>
    punisherFetch<{ studyRequest: StudyRequest }>(`/study-requests/${enc(id)}/resolve`, { method: "POST" }).then(
      (r) => r.studyRequest,
    ),

  // Lectures
  startInstant: (body: InstantRequest) =>
    punisherFetch<InstantResponse>("/lectures/instant", { method: "POST", json: body }),
  getJob: (jobId: string) => punisherFetch<GenerationJob>(`/jobs/${enc(jobId)}`),
  getLecture: (id: string) => punisherFetch<Lecture>(`/lectures/${enc(id)}`),
  saveProgress: (id: string, anchor: string, percent: number) =>
    punisherFetch<{ anchor: string; percent: number }>(`/lectures/${enc(id)}/progress`, {
      method: "PUT",
      json: { anchor, percent },
      keepalive: true,
    }),
  completeLecture: (id: string) =>
    punisherFetch<CompleteResult>(`/lectures/${enc(id)}/complete`, { method: "POST" }),
  sendFeedback: (id: string, feedback: FeedbackKind, anchor: string) =>
    punisherFetch<FeedbackResponse>(`/lectures/${enc(id)}/feedback`, {
      method: "POST",
      json: { feedback, anchor, comment: "" },
    }),
  listQuestions: (lectureId: string) =>
    punisherFetch<{ questions: Question[] }>(`/lectures/${enc(lectureId)}/questions`).then((r) => r.questions),
  ask: (lectureId: string, anchor: string, question: string) =>
    punisherFetch<{ question: Question }>(`/lectures/${enc(lectureId)}/questions`, {
      method: "POST",
      json: { anchor, question },
    }).then((r) => r.question),
  dismissLecture: (id: string) =>
    punisherFetch<{ lectureId: string; dismissed: boolean }>(`/lectures/${enc(id)}/dismiss`, { method: "POST" }),
  restoreLecture: (id: string) =>
    punisherFetch<{ lectureId: string; dismissed: boolean }>(`/lectures/${enc(id)}/dismiss`, { method: "DELETE" }),
  getLectureLinks: (id: string) => punisherFetch<LectureLinks>(`/lectures/${enc(id)}/links`),
  explainSelection: (lectureId: string, anchor: string, selection: string) =>
    punisherFetch<SelectionExplanation>(`/lectures/${enc(lectureId)}/explain`, {
      method: "POST",
      json: { anchor, selection },
    }),
  getQuestion: (id: string) =>
    punisherFetch<{ question: Question }>(`/questions/${enc(id)}`).then((r) => r.question),

  // History (cross-lecture)
  listMyQuestions: (params: { conceptId?: string; limit?: number; cursor?: string } = {}) =>
    punisherFetch<{ questions: MyQuestion[]; nextCursor: string }>(`/me/questions${query(params)}`).then(
      (r): Page<MyQuestion> => ({ items: r.questions, nextCursor: r.nextCursor }),
    ),
  listMyLectures: (params: { status: MyLectureStatus; limit?: number; cursor?: string }) =>
    punisherFetch<{ lectures: MyLecture[]; nextCursor: string }>(`/me/lectures${query(params)}`).then(
      (r): Page<MyLecture> => ({ items: r.lectures, nextCursor: r.nextCursor }),
    ),

  // Learning settings
  getLearningSettings: () => punisherFetch<LearningSettingsResponse>("/me/settings"),
  saveLearningSettings: (patch: LearningSettingsPatch) =>
    punisherFetch<LearningSettingsResponse>("/me/settings", { method: "PUT", json: patch }),

  // Inquiries ('이 질문에 답하기')
  createInquiry: (question: string) => punisherFetch<Inquiry>("/inquiries", { method: "POST", json: { question } }),
  listInquiries: (filter: { open?: boolean; concept?: string } = {}) =>
    punisherFetch<{ inquiries: Inquiry[] }>(
      `/inquiries${query({ open: filter.open ? "1" : undefined, concept: filter.concept })}`,
    ).then((r) => r.inquiries),
  getInquiry: (id: string) => punisherFetch<Inquiry>(`/inquiries/${enc(id)}`),
  regenerateInquiry: (id: string) => punisherFetch<Inquiry>(`/inquiries/${enc(id)}/regenerate`, { method: "POST" }),
  createFollowup: (id: string, body: { question: string; quote?: string }) =>
    punisherFetch<InquiryFollowup>(`/inquiries/${enc(id)}/followups`, { method: "POST", json: body }),
  retryFollowup: (id: string, followupId: string) =>
    punisherFetch<InquiryFollowup>(`/inquiries/${enc(id)}/followups/${enc(followupId)}/retry`, { method: "POST" }),
  listCompleted: (params: { area?: string; q?: string; limit?: number; cursor?: string }) =>
    punisherFetch<{ items: CompletedItem[]; nextCursor: string }>(`/me/completed${query(params)}`).then(
      (r): Page<CompletedItem> => ({ items: r.items, nextCursor: r.nextCursor }),
    ),
  completedAreas: (q?: string) =>
    punisherFetch<{ areas: CompletedArea[] }>(`/me/completed/areas${query({ q })}`).then((r) => r.areas),
  setInquiryRead: (id: string, read: boolean) =>
    punisherFetch<void>(`/inquiries/${enc(id)}/read`, { method: read ? "POST" : "DELETE" }),

  // Next chapter
  chapterSuggestions: () => punisherFetch<ChapterSuggestionsResponse>("/chapters/suggestions"),
  createChapterDraft: (body: { suggestionIds: string[]; freeText: string }) =>
    punisherFetch<ChapterDraftResponse>("/chapters/drafts", { method: "POST", json: body }),
  applyChapterDraft: (draftId: string) =>
    punisherFetch<{ chapterId: string }>(`/chapters/drafts/${enc(draftId)}/apply`, { method: "POST" }),

  // Prompt blocks (학습 설정 > 강의 프롬프트)
  listPrompts: () => punisherFetch<PromptsResponse>("/me/prompts"),
  savePrompt: (id: string, body: string) =>
    punisherFetch<PromptSaveResponse>(`/me/prompts/${enc(id)}`, { method: "PUT", json: { body } }),
  resetPrompt: (id: string) => punisherFetch<PromptSaveResponse>(`/me/prompts/${enc(id)}`, { method: "DELETE" }),
  promptHistory: (id: string) =>
    punisherFetch<{ history: PromptHistoryEntry[] }>(`/me/prompts/${enc(id)}/history`).then((r) => r.history),
  restorePrompt: (id: string, historyId: number) =>
    punisherFetch<PromptSaveResponse>(`/me/prompts/${enc(id)}/restore`, { method: "POST", json: { historyId } }),
  previewPrompt: (id: string, body?: string) =>
    punisherFetch<PromptPreview>(`/me/prompts/${enc(id)}/preview`, {
      method: "POST",
      json: body === undefined ? {} : { body },
    }),
  regenerateLecture: (id: string) =>
    punisherFetch<InstantResponse>(`/lectures/${enc(id)}/regenerate`, { method: "POST" }),

  // 숙달 확인 ('확인하기') — 모든 동작이 CheckDetail 을 돌려준다(pending 이면 202).
  startCheck: (lectureId: string) =>
    punisherFetch<CheckDetail>("/checks", { method: "POST", json: { kind: "check", lectureId } }),
  /** 사전 진단 — 예측 질문은 지금 낼 것이 없으면 204(undefined). 배치 진단할 개념이 부족하면 409 NO_PLACEMENT. */
  startDiagnostic: (body: DiagnosticRequest) =>
    punisherFetch<CheckDetail | undefined>("/checks", { method: "POST", json: body }),
  getCheck: (id: string) => punisherFetch<CheckDetail>(`/checks/${enc(id)}`),
  /** 예측·배치 문항의 보기 또는 '모르겠어요'(피드백 없이 기록). */
  answerChoice: (id: string, answer: DiagnosticChoice) =>
    punisherFetch<CheckDetail>(`/checks/${enc(id)}/answers`, { method: "POST", json: answer }),
  answerCheck: (id: string, text: string) =>
    punisherFetch<CheckDetail>(`/checks/${enc(id)}/answers`, { method: "POST", json: { text } }),
  revealCheck: (id: string) => punisherFetch<CheckDetail>(`/checks/${enc(id)}/reveal`, { method: "POST" }),
  disputeCheck: (id: string, turnId: string, reason?: string) =>
    punisherFetch<CheckDetail>(`/checks/${enc(id)}/turns/${enc(turnId)}/dispute`, {
      method: "POST",
      json: reason ? { reason } : {},
    }),
  retryCheck: (id: string) => punisherFetch<CheckDetail>(`/checks/${enc(id)}/retry`, { method: "POST" }),
  abandonCheck: (id: string) => punisherFetch<CheckDetail>(`/checks/${enc(id)}/abandon`, { method: "POST" }),
  /** 자기 채점 — 답을 보낸 뒤(current.canSelfGrade)에만. */
  selfGradeCheck: (id: string, verdict: "correct" | "partial" | "incorrect") =>
    punisherFetch<CheckDetail>(`/checks/${enc(id)}/self-grade`, { method: "POST", json: { verdict } }),
  listCheckHistory: (params: { kind?: CheckHistoryKind; limit?: number; cursor?: string } = {}) =>
    punisherFetch<CheckHistoryPage>(`/checks/history${query(params)}`),
  // 간격 복습 — 오늘 세션 생성/반환(201·200·202), 할 것이 없으면 409 NO_REVIEWS.
  startReview: () => punisherFetch<CheckDetail>("/reviews", { method: "POST" }),
  getReviewSummary: () => punisherFetch<ReviewSummary>("/reviews/summary"),
  // 개인화(P4) — 확인 결과의 오개념 인정 · 목록
  acknowledgeMisconception: (turnId: string) =>
    punisherFetch<{ misconception: Misconception }>("/misconceptions", { method: "POST", json: { turnId } }).then((r) => r.misconception),
  listMisconceptions: (status: MisconceptionStatus) =>
    punisherFetch<{ misconceptions: Misconception[] }>(`/misconceptions${query({ status })}`).then((r) => r.misconceptions),
  getMasteryMetrics: () => punisherFetch<MasteryMetrics>("/mastery/metrics"),
  /** 여러 개념의 숙달 요약(지도 링, 한 번에 MASTERY_BATCH_MAX 개까지). */
  getMasteryBatch: (conceptIds: string[]) =>
    punisherFetch<{ concepts: ConceptMasterySummary[] }>(`/mastery${query({ conceptIds: conceptIds.join(",") })}`).then((r) => r.concepts),
  getMastery: (conceptId: string) => punisherFetch<ConceptMastery>(`/mastery${query({ conceptId })}`),
  /** 목표 id 에 '#' 가 들어 있어 경로에 넣을 때 인코딩한다. */
  relearnObjective: (objectiveId: string) =>
    punisherFetch<ConceptMastery>(`/objectives/${enc(objectiveId)}/relearn`, { method: "POST" }),

  // Notes
  listNotes: (filter: { lectureId?: string; conceptId?: string } = {}) =>
    punisherFetch<{ notes: Note[] }>(`/notes${query(filter)}`).then((r) => r.notes),
  createNote: (input: NoteInput) => punisherFetch<Note>("/notes", { method: "POST", json: input }),
  deleteNote: (id: string) => punisherFetch<void>(`/notes/${enc(id)}`, { method: "DELETE" }),

  // Goals
  listGoals: () => punisherFetch<{ goals: Goal[] }>("/goals").then((r) => r.goals),
  createGoal: (input: Required<Pick<GoalInput, "title" | "targetConceptIds">>) =>
    punisherFetch<Goal>("/goals", { method: "POST", json: input }),
  updateGoal: (id: string, input: GoalInput) =>
    punisherFetch<Goal>(`/goals/${enc(id)}`, { method: "PATCH", json: input }),
  deleteGoal: (id: string) => punisherFetch<void>(`/goals/${enc(id)}`, { method: "DELETE" }),
  getGoalPath: (id: string) => punisherFetch<GoalPathResponse>(`/goals/${enc(id)}/path`),
};

/** SSE URL for an instant lecture job (consumed with EventSource). */
export function instantStreamUrl(jobId: string): string {
  return `/v1/lectures/instant/${enc(jobId)}/stream`;
}
