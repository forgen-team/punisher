import { useMemo, useRef } from "react";
import { useToast } from "@forgen-team/ui";
import type { ShellCommand } from "@/src/shared/config/runtime";
import { lectureHref, startLectureHref, useStartReview, useToday } from "@/src/entities/punisher";
import { checkHref } from "@/src/features/punisher/check-start";
import { describePunisherError, type Recommendation } from "@/src/shared/api/punisher";
import { useT } from "@/src/shared/i18n";
import { useRouter } from "@/src/shared/lib/router";
import { messages } from "./i18n";

type T = (typeof messages)["ko"];

export interface CommandInput {
  /** 오늘의 다음 강의(첫 추천): 제목과 주소. 없으면 명령을 넣지 않는다 */
  next?: { title: string; href: string };
  startReview: () => void;
}

/** 오늘의 추천 중 첫 강의(이어보기 제외) → "다음 강의" 명령. 강의가 있으면 그 강의, 없으면 개념으로 강의 열기. */
export function nextLecture(items: Recommendation[] | undefined): CommandInput["next"] {
  const next = (items ?? []).find((item) => item.kind !== "resume");
  if (!next) { return undefined; }
  const href = next.lectureId ? lectureHref(next.lectureId) : startLectureHref(next.conceptId);
  return { title: next.conceptName || next.title, href };
}

/**
 * ⌘K 명령(셸·단독판 같은 목록 — 재설계 §3). 셸 안은 셸 통합 검색에(ctx.setCommands), 단독판은 AppFrame 명령 팔레트에 넣는다.
 * 화면 이동(APP-CONTRACT §3 "명령", 재설계 §3 목록). 화면 이동은 href(진짜 링크), 복습 시작은 onSelect.
 * 최대 50개 — 지금은 11개. 모양은 ui CommandItem 과 같다(셸 계약 AppCommand ⊂ CommandItem).
 */
export function buildCommands(t: T, input: CommandInput): ShellCommand[] {
  const list: ShellCommand[] = [
    { id: "today", label: t.cmdToday, keywords: ["today", "오늘", "추천"], href: "/" },
    { id: "map", label: t.cmdMap, keywords: ["map", "지도", "개념"], href: "/map" },
    { id: "history", label: t.cmdHistory, keywords: ["history", "기록"], href: "/history" },
    { id: "review", label: t.cmdReview, group: t.cmdGroupReview, keywords: ["review", "복습"], onSelect: input.startReview },
    { id: "cards", label: t.cmdCards, group: t.cmdGroupReview, keywords: ["cards", "카드", "flashcard"], href: "/cards" },
    { id: "find", label: t.cmdFind, keywords: ["search", "find", "검색", "찾기", "메모", "질문"], href: "/history" },
    { id: "new-chapter", label: t.cmdNewChapter, keywords: ["chapter", "챕터"], href: "/chapters/new" },
    { id: "goals", label: t.cmdGoals, keywords: ["goal", "deadline", "목표", "기한"], href: "/map?panel=goals" },
    { id: "prompts", label: t.cmdPrompts, keywords: ["prompt", "프롬프트"], href: "/settings/prompts" },
    { id: "settings", label: t.appSettings, keywords: ["settings", "설정", "FSRS", "유지율", "알림"], href: "/settings" },
  ];
  if (input.next) {
    list.splice(1, 0, { id: "next-lecture", label: t.cmdNextLecture(input.next.title).slice(0, 60), keywords: ["next", "다음"], href: input.next.href });
  }
  return list;
}

/** 지금 명령 목록 — 다음 강의는 오늘 추천에서, 복습 시작은 세션을 만들어 복습 화면으로(실패하면 토스트). */
export function useAppCommands(): ShellCommand[] {
  const t = useT(messages);
  const router = useRouter();
  const toast = useToast();
  const today = useToday();
  const review = useStartReview();
  const reviewRef = useRef(review);
  reviewRef.current = review;
  const next = nextLecture(today.data?.items);
  const nextTitle = next?.title ?? "";
  const nextHref = next?.href ?? "";
  return useMemo(
    () => buildCommands(t, {
      next: nextHref ? { title: nextTitle, href: nextHref } : undefined,
      startReview: () => reviewRef.current.mutate(undefined, {
        onSuccess: (detail) => router.push(checkHref(detail.session.id)),
        onError: (err) => toast.show(describePunisherError(err) || t.noReviews, { tone: "danger" }),
      }),
    }),
    [t, nextTitle, nextHref, router, toast],
  );
}
