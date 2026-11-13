import type { CheckDetail, CheckItem } from "@/src/shared/api/punisher";

/** 지금 답할 진단 문항(세션이 active 이고 답할 수 있을 때만). */
export function currentDiagnosticItem(detail: CheckDetail | undefined): CheckItem | null {
  if (!detail || detail.session.status !== "active" || detail.pending || !detail.current?.canAnswer) { return null; }
  return detail.items.find((item) => item.id === detail.current?.itemId) ?? null;
}

/** 보기 목록 — choice 문항만(빈 보기는 뺀다). */
export function choicesOf(item: CheckItem): string[] {
  if (item.format !== "choice") { return []; }
  return (item.material?.choices ?? []).filter((c) => c.trim() !== "");
}

/** "n/N" 진행 — 지금 문항의 순번과 전체 수. */
export function diagnosticProgress(detail: CheckDetail): { index: number; total: number } {
  const total = detail.session.itemIds.length;
  return { index: Math.min(detail.session.cursor + 1, total), total };
}

/** 문항 물음 + 자료 코드(있으면 코드 블록). */
export function promptMarkdown(item: CheckItem): string {
  const code = item.material?.code;
  return code ? `${item.prompt}\n\n\`\`\`${item.material?.lang ?? ""}\n${code}\n\`\`\`` : item.prompt;
}
