
import { Button, Section } from "@forgen-team/ui";
import { describeCheckError } from "@/src/entities/punisher";
import { useT } from "@/src/shared/i18n";
import { messages } from "../i18n";

interface PlacementOfferProps {
  starting: boolean;
  error: unknown;
  onStart: () => void;
  onSkip: () => void;
}

/** '이대로 열기' 뒤 배치 진단 — 시작하거나 건너뛰고 지도로 간다. */
export function PlacementOffer({ starting, error, onStart, onSkip }: PlacementOfferProps) {
  const t = useT(messages);
  return (
    <Section variant="card" title={t.placementTitle}>
      <div className="flex flex-wrap justify-end gap-2 pb-4 pt-1">
        <Button variant="ghost" onClick={onSkip} disabled={starting}>
          {t.placementSkip}
        </Button>
        <Button variant="primary" onClick={onStart} loading={starting}>
          {t.placementStart}
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
