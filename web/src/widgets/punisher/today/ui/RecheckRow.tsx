
import { Link } from "@/src/shared/lib/router";
import { ListChecks } from "lucide-react";
import { ListRow } from "@/src/shared/ui/kit";
import { areaLabel, describeCheckError } from "@/src/entities/punisher";
import { useLaunchCheck, useLaunchReview } from "@/src/features/punisher/check-start";
import type { RecheckCard } from "@/src/shared/api/punisher";
import { useT } from "@/src/shared/i18n";
import { messages } from "../i18n";
import { recheckAction } from "../lib/recheck";
import { mapHref } from "./RecommendationRow";

/** "○○ 다시 확인" — 복습 세션(만기) 또는 그 개념 강의의 확인하기로. */
export function RecheckRow({ card }: { card: RecheckCard }) {
  const t = useT(messages);
  const review = useLaunchReview();
  const check = useLaunchCheck(card.lectureId ?? "");
  const action = recheckAction(card);
  const title = t.recheckConcept(card.conceptName);
  const error = review.error ?? check.error;
  const meta = (
    <>
      {areaLabel(card.area)}
      {error && <span role="alert" className="block truncate text-danger">{describeCheckError(error)}</span>}
    </>
  );
  if (action === "map") {
    return <ListRow leading={<ListChecks aria-hidden />} href={mapHref(card.conceptId)} linkAs={Link} title={title} meta={meta} />;
  }
  return (
    <ListRow
      leading={<ListChecks aria-hidden />}
      title={title}
      meta={meta}
      disabled={review.isPending || check.isPending}
      onClick={action === "review" ? review.launch : check.launch}
    />
  );
}
