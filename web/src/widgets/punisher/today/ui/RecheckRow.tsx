
import { Link } from "@/src/shared/lib/router";
import { ListRow } from "@/src/shared/ui/kit";
import { areaLabel, describeCheckError } from "@/src/entities/punisher";
import { useLaunchCheck, useLaunchReview } from "@/src/features/punisher/check-start";
import type { RecheckCard } from "@/src/shared/api/punisher";
import { recheckAction } from "../lib/recheck";
import { mapHref } from "./RecommendationRow";

/** "○○ 다시 확인" — 복습 세션(만기) 또는 그 개념 강의의 확인하기로. */
export function RecheckRow({ card }: { card: RecheckCard }) {
  const review = useLaunchReview();
  const check = useLaunchCheck(card.lectureId ?? "");
  const action = recheckAction(card);
  const title = `${card.conceptName} 다시 확인`;
  const error = review.error ?? check.error;
  const meta = (
    <>
      {areaLabel(card.area)}
      {error && <span role="alert" className="block truncate text-danger">{describeCheckError(error)}</span>}
    </>
  );
  if (action === "map") {
    return <ListRow href={mapHref(card.conceptId)} linkAs={Link} title={title} meta={meta} />;
  }
  return (
    <ListRow
      title={title}
      meta={meta}
      disabled={review.isPending || check.isPending}
      onClick={action === "review" ? review.launch : check.launch}
    />
  );
}
