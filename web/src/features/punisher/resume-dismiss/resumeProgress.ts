import { translate, type Locale } from "@/src/shared/i18n";
import { messages } from "./i18n";

/** 강의는 항상 이 네 부분이다(서버 lecture.Validate 가 개수와 순서를 강제). */
const LECTURE_PARTS = ["situation_question", "principles", "walkthrough", "judgment_next"];

/** 읽던 위치(anchor: "principles" 또는 "principles-3")를 "2/4부분"으로. 위치를 모르면 "". */
export function resumePosition(anchor: string | undefined, locale: Locale): string {
  if (!anchor) { return ""; }
  const index = LECTURE_PARTS.findIndex((kind) => anchor === kind || anchor.startsWith(`${kind}-`));
  return index === -1 ? "" : translate(messages, locale).part(index + 1, LECTURE_PARTS.length);
}
