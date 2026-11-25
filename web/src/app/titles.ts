import type { messages } from "./i18n";

type AppMessages = (typeof messages)["ko"];
/** 화면 제목으로 쓰는 글 키(문구 함수는 빼고). */
export type TitleKey = { [K in keyof AppMessages]: AppMessages[K] extends string ? K : never }[keyof AppMessages];

const TITLES: Array<[RegExp, TitleKey]> = [
  [/^\/map/, "titleMap"],
  [/^\/history/, "titleHistory"],
  [/^\/chapters\/new/, "titleNewChapter"],
  [/^\/lectures\//, "titleLecture"],
  [/^\/inquiries\//, "titleInquiry"],
  [/^\/check\//, "titleCheck"],
  [/^\/instant\//, "titleInstant"],
  [/^\/cards/, "titleCards"],
  [/^\/search/, "titleSearch"],
  [/^\/settings\/prompts/, "titlePrompts"],
  [/^\/settings/, "titleSettings"],
];

/** Screen title for a path (document title standalone, the shell's top bar inside Witems). */
export function titleKeyFor(pathname: string): TitleKey {
  return TITLES.find(([re]) => re.test(pathname))?.[1] ?? "titleToday";
}
