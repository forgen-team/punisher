import type { CheckDetail } from "@/src/shared/api/punisher";

export interface SynthesisRow {
  itemId: string;
  prompt: string;
  result: "passed" | "pulled" | "skipped" | "recorded";
}

/** 종합 문제 결과 줄 — 문항(통합)마다 통과 · 복습 당김 · 기록만. */
export function synthesisRows(detail: CheckDetail): SynthesisRow[] {
  if (detail.session.kind !== "synthesis") { return []; }
  const prompts = new Map(detail.items.map((it) => [it.id, it.prompt]));
  return (detail.session.result?.items ?? []).map((r) => ({
    itemId: r.itemId,
    prompt: prompts.get(r.itemId) ?? "",
    result: SYNTHESIS_RESULT[r.outcome] ?? "recorded",
  }));
}

const SYNTHESIS_RESULT: Partial<Record<string, SynthesisRow["result"]>> = { "unaided_correct": "passed", failed: "pulled", skipped: "skipped" };
