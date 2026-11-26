import type { GraphChapter, NextChapterPrompt } from "./chapterTypes";
import type { MapInquiry } from "./inquiryTypes";
import type { MapRequest, ReaderLevel } from "./learningTypes";
import type { RecheckCard } from "./misconceptionTypes";
import type { TodayReview } from "./reviewTypes";

// DTOs mirrored from apps/punisher-server/internal/{models,learning,services,lecture,store}.
// Keep in sync with the Go json tags; the server rejects unknown request fields.

/** 0단계 기초 트랙(learner-level ②): depth 0, 수준에 따라 기본 숨김. */
export const FOUNDATIONS_AREA = "foundations";

export const PUNISHER_AREAS = [
  FOUNDATIONS_AREA,
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

/** 처음 설정 2단계(영역별 아는 정도)에 묻는 영역 — 0단계 기초는 묻지 않는다. */
export const FAMILIARITY_AREAS = PUNISHER_AREAS.filter(
  (a): a is Exclude<PunisherArea, typeof FOUNDATIONS_AREA> => a !== FOUNDATIONS_AREA,
);
export type FamiliarityArea = (typeof FAMILIARITY_AREAS)[number];

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
  /** 0단계 기초 개념이고 내 수준에서 기본으로 숨김(GET /v1/concepts) */
  hiddenByDefault?: boolean;
  /** '필요 없음'으로 뺐다(이미 앎·완료와 별개 — 선행 잠금은 그대로). */
  dismissed?: boolean;
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
  /** '필요 없음'으로 뺀 이웃(뺀 선행 개념도 여전히 잠금). */
  dismissed?: boolean;
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
  /** 0단계 기초 개념이고 내 수준(실무자·숙련자·미설정)에서 기본으로 접혀 있음 */
  hiddenByDefault?: boolean;
  /** '필요 없음' — 지도에 흐리게, 다음 개념(frontier) 아님. */
  dismissed?: boolean;
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
  /** 내 수준에서 0단계 기초가 기본으로 접혀 있음(노드는 hiddenByDefault) */
  foundationsHidden: boolean;
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
  /** 끝냈지만 확인하기를 안 한 강의의 "○○ 확인하기"(추천·exhausted 와 별개) — 확인해야 복습 일정이 잡힌다 */
  unchecked?: UncheckedCard[];
  /** 이번 주 종합 문제를 지금 할 수 있는지(v3 §5) */
  synthesis?: { due: boolean };
  /** 오늘 학습 세션 시간 합계(v2.2 §5) */
  session?: TodaySession;
}

/** GET /today 의 unchecked — 확인하기를 마치지 않은 완료 강의(다시 만든 강의는 새 버전). */
export interface UncheckedCard {
  lectureId: string;
  conceptId: string;
  conceptName: string;
  area: string;
  title: string;
}

export interface TodaySession {
  lectureMinutes: number;
  checkMinutes: number;
  reviewMinutes: number;
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
  /** '필요 없음' — 경로에는 남는다(다음 단계를 여전히 막는다). */
  dismissed?: boolean;
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

export type InstantStatus = "existing" | "started" | "attached" | "needs_selection" | "already_learned" | "inquiry";

export interface InstantRequest {
  conceptId?: string;
  query?: string;
  depth?: number;
  autoStart?: boolean;
  /** "이 개념 더 깊게" — 결과 강의를 이 강의 아래에 연결한다 */
  parentLectureId?: string;
  parentAnchor?: string;
  /** 학습 요청에서 시작 — 요청 context 가 knowledge_gap 이면 그 빈칸이 강의 입력이 된다 */
  studyRequestId?: string;
}

export interface InstantResponse {
  status: InstantStatus;
  conceptId?: string;
  lectureId?: string;
  jobId?: string;
  streamUrl?: string;
  resolution?: Resolution;
  /** status "inquiry": 읽던 강의와 같은 개념이라 질문 해설로 대신 만든 경우 */
  inquiryId?: string;
}

export type BlockType =
  | "markdown" | "mermaid" | "code" | "log" | "misconception" | "term" | "sources"
  // 구조 블록(v2.2 §6) — data 는 trace·chart·diff, kind 는 keypoint 종류
  | "keypoint" | "trace" | "chart" | "diff" | "pause" | "exercise" | "answer" | "summary"
  // 실습(v3 §4) — kind 는 os(macos | linux | any), text 는 "$ 명령" · 보는 곳: · 예상:
  | "lab"
  // 방금 읽은 걸 떠올려 보기 — text 는 질문 + "떠올릴 것:" 줄, objective 는 목표 id
  | "recall";

export interface LectureBlock {
  anchor: string;
  type: BlockType;
  text?: string;
  lang?: string;
  term?: string;
  conceptId?: string;
  proposed?: boolean;
  sources?: DocumentLink[];
  /** 구조 블록 데이터(JSON 객체) — 화면이 검증한 뒤 그린다 */
  data?: unknown;
  /** keypoint 종류: definition | caution | rule · lab 은 os: macos | linux | any · exercise 는 guided | why | error | free | modify */
  kind?: string;
  /** recall 이 다루는 목표 id */
  objective?: string;
}

export interface LectureSection {
  anchor: string;
  kind: string;
  title: string;
  /** 모델이 쓴 분(표시에는 estimatedMinutes 우선) */
  minutes: number;
  /** 서버 계산 섹션 시간(GET /lectures/{id}, v2.2 §5) — 섹션마다 반올림해 합이 총시간과 1~2분 다를 수 있다 */
  estimatedMinutes?: number;
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

export type LectureReviewState = "unreviewed" | "passed" | "flagged";

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
  /** flagReason 의 사유 code(서버 internal/i18n) — 있으면 화면 언어로 옮겨 보인다 */
  flagReasonCodes?: string[];
  /** 검수 상태(v2.2) — 없으면 예전 응답(flagged·status 로 판단) */
  reviewState?: LectureReviewState;
  /** 생성에 쓴 프롬프트 버전 — 내 프롬프트를 썼으면 '+u:<해시8>' 가 붙는다 */
  promptVersion?: string;
  /** 이 강의를 쓴 설명 수준 — 없으면 예전 응답(practitioner 로 본다) */
  readerLevel?: ReaderLevel;
  /** 이 강의를 쓴 AI(claude|codex|agy) — 없으면 예전 응답(claude) */
  provider?: string;
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
  /** 읽기 스레드(④): 고른 구절 — 예전 질문·재설명 요청은 "" */
  quote?: string;
  /** 이어 묻기면 루트 질문 id, 루트면 "" */
  parentId?: string;
  mode?: ThreadMode;
}

/** 읽기 스레드 요청 종류 — easier·deeper 는 구절만으로, question 은 질문 문장이 필요하다. */
export type ThreadMode = "easier" | "deeper" | "question";

/** POST /lectures/{id}/threads */
export interface ThreadStartInput {
  anchor: string;
  quote: string;
  mode: ThreadMode;
  question?: string;
}

/** POST /questions/{id}/followups — mode easier·deeper 는 "직전 답을 더 쉽게/깊게" */
export interface ThreadFollowupInput {
  question?: string;
  mode?: ThreadMode;
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
  startedAt?: string;
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
