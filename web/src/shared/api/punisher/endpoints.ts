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
import type { ExampleLanguage, LearningSettingsResponse, SelectionExplanation } from "./learningTypes";

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
  markKnown: (id: string) =>
    punisherFetch<{ conceptId: string; state: string }>(`/concepts/${enc(id)}/known`, { method: "POST" }),
  getGloss: (conceptId: string) =>
    punisherFetch<{ conceptId: string; gloss: string; cached: boolean }>(`/terms/${enc(conceptId)}/gloss`),

  // Today & study requests
  getToday: (limit = 5) => punisherFetch<TodayResponse>(`/today${query({ limit })}`),
  listStudyRequests: (limit = 20) =>
    punisherFetch<{ studyRequests: StudyRequest[] }>(`/study-requests${query({ limit })}`).then(
      (r) => r.studyRequests,
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
  saveLearningSettings: (exampleLanguage: ExampleLanguage) =>
    punisherFetch<LearningSettingsResponse>("/me/settings", { method: "PUT", json: { exampleLanguage } }),

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
