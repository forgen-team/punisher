
import { Section } from "@/src/shared/ui/kit";
import type { CheckDetail } from "@/src/shared/api/punisher";
import { Markdown } from "@/src/shared/ui/punisher/Markdown";
import { currentDiagnosticItem, diagnosticProgress, promptMarkdown } from "../lib/diagnostic";
import { DiagnosticAnswer } from "./DiagnosticAnswer";
import { DIAGNOSTIC_LABELS } from "./constants";

/** 강의 위 예측 질문(피드백 없이 기록만) — 답할 문항이 있을 때만 보인다. */
export function PretestPanel({ detail }: { detail: CheckDetail | undefined }) {
  const item = currentDiagnosticItem(detail);
  if (!detail || detail.session.kind !== "pretest" || !item) { return null; }
  const { index, total } = diagnosticProgress(detail);
  return (
    <Section variant="card" title={DIAGNOSTIC_LABELS.pretest} description={`${index}/${total}`} className="mt-4">
      <div className="flex flex-col gap-3 pb-4 pt-1">
        <Markdown className="text-base">{promptMarkdown(item)}</Markdown>
        <DiagnosticAnswer key={item.id} detail={detail} item={item} />
      </div>
    </Section>
  );
}
