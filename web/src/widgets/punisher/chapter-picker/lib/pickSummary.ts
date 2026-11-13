import { PICK_BAR_LABELS } from "../ui/constants";

/** 하단 고정 바의 선택 상태 — "선택 n개" (+ " · 직접 입력"). 아무것도 없으면 "". */
export function pickSummary(selectedCount: number, freeText: string): string {
  return [selectedCount > 0 ? PICK_BAR_LABELS.picked(selectedCount) : "", freeText.trim() ? PICK_BAR_LABELS.freeText : ""]
    .filter(Boolean)
    .join(" · ");
}
