import type { LectureSection } from "./types";

/** '이 질문에 답하기' 해설 상태 — pending: 쓰는 중(다시 만들기 중이면 이전 해설이 함께 온다). */
export type InquiryStatus = "pending" | "answered" | "failed";

/** 해설에 나온 개념 — related: 지도에 있던 개념, proposed: 해설이 지도에 새로 제안한 개념. */
export interface InquiryConcept {
  id: string;
  name: string;
  area: string;
  kind: "related" | "proposed";
  userState: string;
}

/** 해설 본문 — 강의와 같은 섹션·블록 구조(같은 렌더러). 섹션: summary · causes · assumptions · verify · fix */
export interface InquiryContent {
  title: string;
  sections: LectureSection[];
}

export interface Inquiry {
  id: string;
  question: string;
  status: InquiryStatus;
  /** 목록 응답에는 비어 있다({}) */
  content?: Partial<InquiryContent>;
  conceptIds: string[];
  proposedConceptIds: string[];
  /** 상세 응답에서만 채워진다 */
  concepts?: InquiryConcept[];
  model: string;
  promptVersion: string;
  error?: string;
  /** '다 봤어요' — 학습 홈 목록에서 빠진다 */
  readAt?: string;
  createdAt: string;
  updatedAt: string;
  answeredAt?: string;
  /** 이어서 묻기(순서대로, 상세 응답에서만) */
  followups?: InquiryFollowup[];
  followupCount: number;
}

/** 이어서 묻기 한 건 — 답은 강의 블록 구조의 섹션 하나. quote: 해설에서 고른 구절('여기 더 자세히'). */
export interface InquiryFollowup {
  id: string;
  question: string;
  quote?: string;
  status: InquiryStatus;
  section?: LectureSection;
  error?: string;
  createdAt: string;
  answeredAt?: string;
}

/** 스레드당 후속 질문 수·길이 상한(서버와 같은 값). */
export const INQUIRY_MAX_FOLLOWUPS = 20;
export const INQUIRY_MAX_FOLLOWUP_CHARS = 2000;
export const INQUIRY_MAX_QUOTE_CHARS = 1000;

/** 완료 목록 항목 — 완료한 강의 또는 답을 받은 질문 스레드. */
export interface CompletedItem {
  kind: "lecture" | "inquiry";
  id: string;
  title: string;
  conceptId?: string;
  conceptName?: string;
  area: string;
  depth?: number;
  lectureKind?: string;
  parentLectureId?: string;
  followupCount?: number;
  lastActivityAt: string;
}

export interface CompletedArea {
  area: string;
  count: number;
}

/** 지도 노드에 붙는 내 질문(답한 것만). */
export interface MapInquiry {
  id: string;
  question: string;
}
