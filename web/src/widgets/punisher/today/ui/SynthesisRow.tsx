import { ListRow } from "@/src/shared/ui/kit";
import { describeCheckError } from "@/src/entities/punisher";
import { useLaunchSynthesis } from "@/src/features/punisher/check-start";

/** "이번 주 종합 문제" — 확인한 개념 여러 개를 한 장면에서 함께 쓰는 문제(주 1회, 누르면 그때 만든다). */
export function SynthesisRow() {
  const synthesis = useLaunchSynthesis();
  const meta = synthesis.error ? (
    <span role="alert" className="block truncate text-danger">{describeCheckError(synthesis.error)}</span>
  ) : undefined;
  return <ListRow title="이번 주 종합 문제" meta={meta} disabled={synthesis.isPending} onClick={synthesis.launch} />;
}
