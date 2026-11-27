import { ListChecks } from "lucide-react";
import { ListRow } from "@forgen-team/ui";
import { areaLabel, describeCheckError } from "@/src/entities/punisher";
import { isLectureNotReviewed, useLaunchCheck } from "@/src/features/punisher/check-start";
import type { UncheckedCard } from "@/src/shared/api/punisher";
import { useT } from "@/src/shared/i18n";
import { messages } from "../i18n";

/** "○○ 확인하기" — 끝냈지만 확인하지 않은 강의의 확인하기(여기서 푼 결과로 복습 일정이 잡힌다). */
export function UncheckedRow({ card }: { card: UncheckedCard }) {
  const t = useT(messages);
  const check = useLaunchCheck(card.lectureId);
  const error = check.error;
  const meta = (
    <>
      {areaLabel(card.area)}
      {error && isLectureNotReviewed(error) && <span role="status" className="block truncate">{t.notReviewed}</span>}
      {error && !isLectureNotReviewed(error) && <span role="alert" className="block truncate text-danger">{describeCheckError(error)}</span>}
    </>
  );
  return <ListRow leading={<ListChecks aria-hidden />} title={t.checkConcept(card.conceptName)} meta={meta} onClick={() => { if (!check.isPending) { check.launch(); } }} />;
}
