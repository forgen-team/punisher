import type { messages } from "./i18n";

export type TitleKey = keyof (typeof messages)["ko"];

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
