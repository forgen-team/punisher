import { translate, type Locale } from "@/src/shared/i18n";
import { errorCodeMessages, reasonMessages } from "./i18n";

type ReasonCode = keyof typeof reasonMessages.ko;
type ErrorCodeWithMessage = keyof typeof errorCodeMessages.ko;

/**
 * 서버가 준 사유 code(stage 이벤트 reasonCodes, 강의 flagReasonCodes)를 화면 언어로 옮긴다.
 * 모르는 code 가 하나라도 있으면 빈 배열 — 호출하는 쪽은 서버 문구(reasons·flagReason)를 그대로 쓴다.
 */
export function translateReasonCodes(codes: unknown, locale?: Locale): string[] {
  if (!Array.isArray(codes) || codes.length === 0) { return []; }
  const t = translate(reasonMessages, locale);
  const out: string[] = [];
  for (const code of codes) {
    if (typeof code !== "string" || !(code in t)) { return []; }
    out.push(t[code as ReasonCode]);
  }
  return out;
}

/** 서버 오류 code 의 화면 문구(카탈로그에 없으면 undefined). */
export function errorCodeMessage(code: string, locale?: Locale): string | undefined {
  const t = translate(errorCodeMessages, locale);
  return code in t ? t[code as ErrorCodeWithMessage] : undefined;
}
