import { getLocale, translate, type Locale } from "@/src/shared/i18n";
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

/**
 * 서버가 저장한 실패 문구(질문 답·대체 설명의 error)를 화면 언어로: 아는 사유 문구면 그 code 의
 * 번역, 모르는 문구는 한국어 화면이면 그대로, 아니면 fallback(서버 원문이 한국어·내부 오류일 수 있어서).
 */
export function localizeServerText(text: string | undefined, fallback: string, locale: Locale = getLocale()): string {
  const trimmed = text?.trim();
  if (!trimmed) { return fallback; }
  const ko = reasonMessages.ko as Record<string, string>;
  const code = Object.keys(ko).find((key) => ko[key] === trimmed);
  if (code) { return translateReasonCodes([code], locale)[0] ?? fallback; }
  return locale === "ko" ? trimmed : fallback;
}
