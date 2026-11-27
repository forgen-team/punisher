import type { messages } from "./i18n";

type AppMessages = (typeof messages)["ko"];
/** 화면 제목으로 쓰는 글 키(문구 함수는 빼고). */
export type TitleKey = { [K in keyof AppMessages]: AppMessages[K] extends string ? K : never }[keyof AppMessages];

const TITLES: Array<[RegExp, TitleKey]> = [
  [/^\/?$/, "titleToday"],
  [/^\/map(\/|$)/, "titleMap"],
  [/^\/history(\/|$)/, "titleHistory"],
  [/^\/chapters\/new$/, "titleNewChapter"],
  [/^\/lectures\/[^/]+$/, "titleLecture"],
  [/^\/inquiries\/[^/]+$/, "titleInquiry"],
  [/^\/check\/[^/]+$/, "titleCheck"],
  [/^\/instant\/[^/]+$/, "titleInstant"],
  [/^\/cards$/, "titleCards"],
  [/^\/start$/, "titleStart"],
  [/^\/settings\/prompts$/, "titlePrompts"],
  [/^\/settings$/, "titleSettings"],
];

/**
 * 경로별 화면 제목 — 화면이 구체적인 제목(강의 제목 등, useScreenTitle)을 주지 않을 때 쓴다.
 * 단독판 Page h1·문서 제목, 셸 안 ctx.setTitle 이 같은 값을 쓴다. 모르는 경로는 "찾을 수 없어요".
 */
export function titleKeyFor(pathname: string): TitleKey {
  return TITLES.find(([re]) => re.test(pathname))?.[1] ?? "titleNotFound";
}
