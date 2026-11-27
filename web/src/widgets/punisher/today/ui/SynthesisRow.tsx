import { ListChecks } from "lucide-react";
import { ListRow } from "@forgen-team/ui";
import { describeCheckError } from "@/src/entities/punisher";
import { useLaunchSynthesis } from "@/src/features/punisher/check-start";
import { useT } from "@/src/shared/i18n";
import { messages } from "../i18n";

/** "이번 주 종합 문제" — 확인한 개념 여러 개를 한 장면에서 함께 쓰는 문제(주 1회, 누르면 그때 만든다). */
export function SynthesisRow() {
  const t = useT(messages);
  const synthesis = useLaunchSynthesis();
  const meta = synthesis.error ? (
    <span role="alert" className="block truncate text-danger">{describeCheckError(synthesis.error)}</span>
  ) : undefined;
  return <ListRow leading={<ListChecks aria-hidden />} title={t.synthesis} meta={meta} onClick={() => { if (!synthesis.isPending) { synthesis.launch(); } }} />;
}
