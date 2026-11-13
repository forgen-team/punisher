
import type { CheckItem, CheckTurn } from "@/src/shared/api/punisher";
import { BlockRenderer } from "../../lecture-viewer";
import { CheckTurnItem } from "./CheckTurnItem";
import { ItemReview } from "./ItemReview";

interface CheckItemCardProps {
  index: number;
  total: number;
  item: CheckItem;
  /** 이 문항의 턴(첫 물음 제외, seq 순) */
  turns: CheckTurn[];
  /** 판정 턴의 놓친 기준 */
  missing: string[];
  anchorHref: string | null;
  anchorLabel: string;
  retryTurnId: string | null;
  onRetry: () => void;
  retrying: boolean;
  onDispute?: () => void;
  disputing: boolean;
}

/** 문항 하나: 물음(prompt) + 자료(material, 강의 코드 블록) → 문답 → 끝났으면 판정·모범답. 이의 문답은 판정 아래. */
export function CheckItemCard({ index, total, item, turns, missing, anchorHref, anchorLabel, retryTurnId, onRetry, retrying, onDispute, disputing }: CheckItemCardProps) {
  const dialogue = turns.filter((t) => t.move !== "dispute");
  const disputes = turns.filter((t) => t.move === "dispute");
  const renderTurn = (t: CheckTurn) => (
    <CheckTurnItem key={t.id} turn={t} onRetry={t.id === retryTurnId ? onRetry : undefined} retrying={retrying} />
  );
  return (
    <section aria-label={`문항 ${index}/${total}`} className="flex flex-col gap-3 border-t border-line pt-4 first:border-t-0">
      <p className="text-xs font-medium text-fg-muted">
        문항 {index}/{total}
      </p>
      <div className="text-base">
        <BlockRenderer block={{ anchor: "", type: "markdown", text: item.prompt }} />
        {item.material?.code && <BlockRenderer block={{ anchor: "", type: "code", text: item.material.code, lang: item.material.lang }} />}
      </div>
      {dialogue.length > 0 && <ul className="flex flex-col gap-3">{dialogue.map((t) => renderTurn(t))}</ul>}
      {item.done && (
        <ItemReview
          item={item}
          missing={missing}
          anchorHref={anchorHref}
          anchorLabel={anchorLabel}
          onDispute={onDispute}
          disputing={disputing}
        />
      )}
      {disputes.length > 0 && <ul className="flex flex-col gap-3">{disputes.map((t) => renderTurn(t))}</ul>}
    </section>
  );
}
