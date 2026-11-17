import { translate, type Locale } from "@/src/shared/i18n";
import { messages } from "../i18n";

/** 하단 고정 바의 선택 상태 — "선택 n개" (+ " · 직접 입력"). 아무것도 없으면 "". */
export function pickSummary(selectedCount: number, freeText: string, locale: Locale): string {
  const t = translate(messages, locale);
  return [selectedCount > 0 ? t.picked(selectedCount) : "", freeText.trim() ? t.freeText : ""]
    .filter(Boolean)
    .join(" · ");
}
