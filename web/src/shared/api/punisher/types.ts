import type { GraphChapter, NextChapterPrompt } from "./chapterTypes";
import type { MapInquiry } from "./inquiryTypes";
import type { MapRequest } from "./learningTypes";
import type { RecheckCard } from "./misconceptionTypes";
import type { TodayReview } from "./reviewTypes";

// DTOs mirrored from apps/punisher-server/internal/{models,learning,services,lecture,store}.
// Keep in sync with the Go json tags; the server rejects unknown request fields.

export const PUNISHER_AREAS = [
  "data-structures",
  "computer-architecture",
  "os",
  "network-web",
  "database",
  "distributed",
  "infra-ops",
  "security-design",
] as const;
export type PunisherArea = (typeof PUNISHER_AREAS)[number];

export type UserConceptState = "not_started" | "in_progress" | "completed" | "known";

export interface DocumentLink {
  title: string;
  url: string;
}

export interface Concept {
  id: string;
  name: string;
  area: string;
  depth: number;
  summary: string;
  practicalCases: string[] | null;
  docLinks: DocumentLink[] | null;
  status: "active" | "proposed" | "retired";
  origin: "seed" | "dynamic";
  curriculumVersion?: string;
  objectiveCount: number;
  userState: UserConceptState;
  updatedAt: string;
}

export interface Objective {
  id: string;
  text: string;
  depth: number;
  ordinal: number;
  userState: string;
}

export interface ConceptLink {
  id: string;
  name: string;
  area: string;
  kind: string;
  userState: UserConceptState;
}

export interface ConceptDetail extends Concept {
  aliases: string[];
  objectives: Objective[];
  prerequisites: ConceptLink[];
  dependents: ConceptLink[];
  related: ConceptLink[];
  locked: boolean;
}

export interface GraphNode {
  id: string;
  name: string;
  area: string;
  depth: number;
  status: string;
  state: UserConceptState;
  locked: boolean;
  frontier: boolean;
  unlocks: number;
  /** 이 개념으로 연결된 대기 학습 요청(개념을 마치면 비어 있음) */
  requests?: MapRequest[];
  /** 이 개념에 대한 내 질문('이 질문에 답하기' 해설) */
  inquiries?: MapInquiry[];
  /** 내 챕터 개념이면 그 챕터 id */
  chapterId?: string;
}

export interface GraphEdge {
  from: string;
  to: string;
  kind: "prerequisite" | "related" | "part_of";
}

export interface AreaCoverage {
  area: string;
  total: number;
  completed: number;
  known: number;
  inProgress: number;
  requested: number;
  dismissed: number;
}

export interface Graph {
  curriculumVersion: string;
  nodes: GraphNode[];
  edges: GraphEdge[];
  coverage: AreaCoverage[];
  /** 아직 개념에 연결되지 않은(분류 중) 학습 요청 */
  pendingRequests: MapRequest[];
  /** 내 챕터(새 챕터: <이름> 묶음) */
  chapters: GraphChapter[];
}

export interface FrontierItem {
  id: string;
  name: string;
  area: string;
  depth: number;
  summary: string;
  state: UserConceptState;
  unlocks: number;
}

export type StudyRequestStatus = "unresolved" | "resolved" | "in_progress" | "done" | "dismissed";

export interface StudyRequest {
  id: string;
  source: "rss" | "manual" | "guardian";
  sourceRef: string;
  query: string;
  context: unknown;
  resolvedConceptIds: string[] | null;
  status: StudyRequestStatus;
  idempotencyKey: string;
  startingConceptId?: string;
  resolveAttempts: number;
  resolveError?: string;
  resolvedAt?: string;
  learned: boolean;
  /** 지금 이 요청의 개념 찾기가 진행 중인지(서버 계산) */
  resolving?: boolean;
  createdAt: string;
  updatedAt: string;
}

export type RecommendationKind = "resume" | "goal_path" | "study_request" | "essential_gap" | "new_goal";

export interface Recommendation {
  kind: RecommendationKind;
  conceptId: string;
  conceptName: string;
  area: string;
  depth: number;
  lectureId?: string;
  goalId?: string;
  studyRequestId?: string;
  title: string;
  reason: string;
  practicalSituation: string;
  estimatedMinutes: number;
  anchor?: string;
  percent?: number;
  /** (이어보기) "더 깊게"로 이 강의를 연 원 강의 */
  parentLectureId?: string;
}

export interface TodayResponse {
  items: Recommendation[];
  exhausted: boolean;
  /** 복습 큐 요약(추천·exhausted 와 별개) */
  review?: TodayReview;
  message?: string;
  /** 남은 학습 분량이 하루 학습량 × 14일보다 적으면 '다음 챕터 고르기' */
  nextChapter?: NextChapterPrompt;
  /** 목표 경로 카드 앞의 "○○ 다시 확인"(추천·exhausted 와 별개) */
  recheck?: RecheckCard[];
}

export interface PathStep {
  conceptId: string;
  name: string;
  area: string;
  depth: number;
  state: UserConceptState;
  locked: boolean;
  isTarget: boolean;
  isNext: boolean;
}

export interface Resolution {
  query: string;
  method: "alias" | "llm" | "proposed" | "none";
  conceptIds: string[] | null;
  proposedConceptId?: string;
  startingConceptId: string;
  path: PathStep[] | null;
  error?: string;
}

export type InstantStatus = "existing" | "started" | "attached" | "needs_selection" | "already_learned";

export interface InstantRequest {
  conceptId?: string;
  query?: string;
  depth?: number;
  autoStart?: boolean;
  /** "이 개념 더 깊게" — 결과 강의를 이 강의 아래에 연결한다 */
  parentLectureId?: string;
  parentAnchor?: string;
}

export interface InstantResponse {
  status: InstantStatus;
  conceptId?: string;
  lectureId?: string;
  jobId?: string;
  streamUrl?: string;
  resolution?: Resolution;
}

export type BlockType = "markdown" | "mermaid" | "code" | "log" | "misconception" | "term" | "sources";

export interface LectureBlock {
  anchor: string;
  type: BlockType;
  text?: string;
  lang?: string;
  term?: string;
  conceptId?: string;
  proposed?: boolean;
  sources?: DocumentLink[];
}

export interface LectureSection {
  anchor: string;
  kind: string;
  title: string;
  minutes: number;
  blocks: LectureBlock[];
}

export interface LectureContent {
  schemaVersion: number;
  title: string;
  conceptId: string;
  objectiveIds: string[];
  depth: number;
  estimatedMinutes: number;
  newScope: string[];
  prerequisitesReferenced: string[];
  sections: LectureSection[];
}

export interface LectureProgress {
  anchor: string;
  percent: number;
  startedAt: string;
  lastSeenAt: string;
  completedAt?: string;
}

export interface LectureReview {
  id: string;
  verdict: "pass" | "fail";
  summary: string;
  attempt: number;
  reviewer: string;
  createdAt: string;
}

export interface Lecture {
  id: string;
  conceptId: string;
  objectiveIds: string[];
  depth: number;
  kind: "library" | "instant";
  status: "draft" | "needs_revision" | "published" | "archived";
  version: number;
  title: string;
  content?: LectureContent;
  sources: DocumentLink[] | null;
  flagged: boolean;
  flagReason?: string;
  /** 생성에 쓴 프롬프트 버전 — 내 프롬프트를 썼으면 '+u:<해시8>' 가 붙는다 */
  promptVersion?: string;
  estimatedMinutes: number;
  createdAt: string;
  publishedAt?: string;
  progress?: LectureProgress;
  reviews?: LectureReview[];
}

export interface CompleteResult {
  conceptId: string;
  objectivesCompleted: string[] | null;
  conceptState: UserConceptState;
}

export type FeedbackKind = "understood" | "insufficient" | "too_easy" | "too_hard" | "want_more";

export interface FeedbackResponse {
  id: string;
  feedback: FeedbackKind;
  altRequestId?: string;
  altRequestStatus?: QuestionStatus;
  altRequestError?: string;
}

export type QuestionStatus = "pending" | "answered" | "failed";

export interface ProposedConcept {
  id: string;
  name: string;
  area: string;
  summary: string;
}

export interface Question {
  id: string;
  lectureId?: string;
  conceptId?: string;
  anchor: string;
  kind: "question" | "alt_explanation";
  question: string;
  answer: string;
  status: QuestionStatus;
  error?: string;
  linkedConceptIds: string[] | null;
  proposedConcepts: ProposedConcept[] | null;
  createdAt: string;
  answeredAt?: string;
}

export interface Note {
  id: string;
  lectureId?: string;
  conceptId?: string;
  anchor: string;
  kind: "note" | "highlight";
  quote: string;
  body: string;
  createdAt: string;
  updatedAt: string;
}

export interface NoteInput {
  lectureId?: string;
  conceptId?: string;
  anchor?: string;
  kind?: "note" | "highlight";
  quote?: string;
  body?: string;
}

export type GoalStatus = "active" | "achieved" | "archived";

export interface Goal {
  id: string;
  title: string;
  targetConceptIds: string[];
  status: GoalStatus;
  createdAt: string;
  updatedAt: string;
}

export interface GoalInput {
  title?: string;
  targetConceptIds?: string[];
  status?: GoalStatus;
}

export interface GoalTargetPath {
  targetConceptId: string;
  startingConceptId: string;
  completed: boolean;
  steps: PathStep[] | null;
}

export interface GoalPathResponse {
  goal: Goal;
  paths: GoalTargetPath[];
}

export interface GenerationJob {
  id: string;
  kind: string;
  conceptId: string;
  depth?: number;
  status: "pending" | "running" | "succeeded" | "failed" | "skipped";
  reason: string;
  error?: string;
  lectureId?: string;
}

export interface MyQuestion extends Question {
  lectureTitle: string;
  conceptName: string;
}

export interface MyLecture {
  id: string;
  conceptId: string;
  conceptName: string;
  title: string;
  depth: number;
  kind: "library" | "instant";
  estimatedMinutes: number;
  percent: number;
  startedAt: string;
  lastSeenAt: string;
  completedAt?: string;
  /** "더 깊게"로 이 강의를 연 원 강의 */
  parentLectureId?: string;
}

export type MyLectureStatus = "completed" | "in_progress";

export interface ConceptLecture {
  id: string;
  title: string;
  depth: number;
  kind: "library" | "instant";
  version: number;
  estimatedMinutes: number;
  publishedAt?: string;
  progressState: UserConceptState;
  percent: number;
}

export interface Page<T> {
  items: T[];
  nextCursor: string;
}
