import type { ShellCommand } from "@/src/shared/config/runtime";
import type { messages } from "@/src/app/i18n";

type T = (typeof messages)["ko"];

export interface CommandInput {
  /** 오늘의 다음 강의(첫 추천): 제목과 주소. 없으면 명령을 넣지 않는다 */
  next?: { title: string; href: string };
  startReview: () => void;
}

/**
 * 셸 ⌘K 검색에 넣는 학습 명령(APP-CONTRACT §3 "명령"). 화면 이동은 href(진짜 링크), 복습 시작은 onSelect.
 * 최대 50개 — 지금은 12개 안팎.
 */
export function buildCommands(t: T, input: CommandInput): ShellCommand[] {
  const list: ShellCommand[] = [
    { id: "today", label: t.cmdToday, keywords: ["today", "오늘", "추천"], href: "/" },
    { id: "map", label: t.cmdMap, keywords: ["map", "지도", "개념"], href: "/map" },
    { id: "history", label: t.cmdHistory, keywords: ["history", "기록"], href: "/history" },
    { id: "review", label: t.cmdReview, group: t.cmdGroupReview, keywords: ["review", "복습"], onSelect: input.startReview },
    { id: "cards", label: t.cmdCards, group: t.cmdGroupReview, keywords: ["cards", "카드", "flashcard"], href: "/cards" },
    { id: "search", label: t.cmdSearch, keywords: ["search", "검색", "메모"], href: "/search" },
    { id: "new-chapter", label: t.cmdNewChapter, keywords: ["chapter", "챕터"], href: "/chapters/new" },
    { id: "goals", label: t.cmdGoals, keywords: ["goal", "deadline", "목표", "기한"], href: "/map?panel=goals" },
    { id: "prompts", label: t.cmdPrompts, keywords: ["prompt", "프롬프트"], href: "/settings/prompts" },
    { id: "settings", label: t.learnSettings, keywords: ["settings", "설정", "FSRS", "유지율", "알림"], href: "/settings" },
  ];
  if (input.next) {
    list.splice(1, 0, { id: "next-lecture", label: t.cmdNextLecture(input.next.title).slice(0, 60), keywords: ["next", "다음"], href: input.next.href });
  }
  return list;
}
