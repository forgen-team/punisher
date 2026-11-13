
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

/** 결과 한 줄의 드러난 오개념 — 오개념마다 '맞아요, 그렇게 알았어요'(누르면 '인정함'). */
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
            <Button
              size="sm"
              variant="secondary"
              disabled={state === "acknowledged" || ack.isPending}
              loading={state === "pending"}
              onClick={() => ack.mutate(m.turnId)}
            >
              {state === "acknowledged" ? MISCONCEPTION_LABELS.acknowledged : MISCONCEPTION_LABELS.acknowledge}
            </Button>
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
