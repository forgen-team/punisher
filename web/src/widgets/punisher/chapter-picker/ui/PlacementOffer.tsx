
import { Button, Section } from "@/src/shared/ui/kit";
import { describeCheckError } from "@/src/entities/punisher";
import { PLACEMENT_OFFER_LABELS } from "./constants";

interface PlacementOfferProps {
  starting: boolean;
  error: unknown;
  onStart: () => void;
  onSkip: () => void;
}

/** '이대로 열기' 뒤 배치 진단 — 시작하거나 건너뛰고 지도로 간다. */
export function PlacementOffer({ starting, error, onStart, onSkip }: PlacementOfferProps) {
  return (
    <Section variant="card" title={PLACEMENT_OFFER_LABELS.title}>
      <div className="flex flex-wrap justify-end gap-2 pb-4 pt-1">
        <Button variant="ghost" onClick={onSkip} disabled={starting}>
          {PLACEMENT_OFFER_LABELS.skip}
        </Button>
        <Button onClick={onStart} loading={starting}>
          {PLACEMENT_OFFER_LABELS.start}
        </Button>
      </div>
      {Boolean(error) && (
        <p role="alert" className="pb-3 text-sm text-danger">
          {describeCheckError(error)}
        </p>
      )}
    </Section>
  );
}
