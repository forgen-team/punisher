
import { Check } from "lucide-react";
import { Button } from "@/src/shared/ui/kit";
import { describeCheckError, useAcknowledgeMisconception } from "@/src/entities/punisher";
import { misconceptionButtonState, type RevealedMisconception } from "../lib/checkView";
import { MISCONCEPTION_LABELS } from "./constants";

interface MisconceptionListProps {
  checkId: string;
  misconceptions: RevealedMisconception[];
  /** 이 세션에서 이미 인정한 턴 */
  acknowledged: ReadonlySet<string>;
}

/** 결과 한 줄의 드러난 오개념 — 오개념마다 '맞아요, 그렇게 알았어요'(인정하면 버튼 대신 체크 + '인정함' 글자). */
export function MisconceptionList({ checkId, misconceptions, acknowledged }: MisconceptionListProps) {
  const ack = useAcknowledgeMisconception(checkId);
  if (misconceptions.length === 0) { return null; }
  const pendingTurnId = ack.isPending ? (ack.variables ?? null) : null;
  return (
    <div className="flex flex-col gap-1.5">
      {misconceptions.map((m) => {
        const state = misconceptionButtonState(m.turnId, acknowledged, pendingTurnId);
        return (
          <div key={m.turnId} className="flex flex-wrap items-center gap-x-2 gap-y-1">
            <p className="break-keep text-fg-secondary">
              {MISCONCEPTION_LABELS.revealed}: {m.text}
            </p>
            {state === "acknowledged" ? (
              <span className="inline-flex items-center gap-1 text-xs text-fg-secondary">
                <Check className="size-3.5 text-success" aria-hidden="true" />
                {MISCONCEPTION_LABELS.acknowledged}
              </span>
            ) : (
              <Button size="sm" variant="secondary" disabled={ack.isPending} loading={state === "pending"} onClick={() => ack.mutate(m.turnId)}>
                {MISCONCEPTION_LABELS.acknowledge}
              </Button>
            )}
          </div>
        );
      })}
      {ack.error && (
        <p role="alert" className="text-sm text-danger">
          {describeCheckError(ack.error)}
        </p>
      )}
    </div>
  );
}
