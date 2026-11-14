import { ListRow } from "@/src/shared/ui/kit";
import { areaLabel, describeCheckError } from "@/src/entities/punisher";
import { isLectureNotReviewed, useLaunchCheck } from "@/src/features/punisher/check-start";
import type { UncheckedCard } from "@/src/shared/api/punisher";

/** "○○ 확인하기" — 끝냈지만 확인하지 않은 강의의 확인하기(여기서 푼 결과로 복습 일정이 잡힌다). */
export function UncheckedRow({ card }: { card: UncheckedCard }) {
  const check = useLaunchCheck(card.lectureId);
  const error = check.error;
  const meta = (
    <>
      {areaLabel(card.area)}
      {error && isLectureNotReviewed(error) && <span role="status" className="block truncate">검수 후 확인할 수 있어요</span>}
      {error && !isLectureNotReviewed(error) && <span role="alert" className="block truncate text-danger">{describeCheckError(error)}</span>}
    </>
  );
  return <ListRow title={`${card.conceptName} 확인하기`} meta={meta} disabled={check.isPending} onClick={check.launch} />;
}
