
import { ChevronRight } from "lucide-react";
import { Badge, List, ListRow } from "@/src/shared/ui/kit";
import type { PromptBlock } from "@/src/shared/api/punisher";
import { STAGE_ORDER } from "./constants";

interface PromptBlockListProps {
  blocks: PromptBlock[];
  selectedId: string | null;
  onSelect: (id: string) => void;
}

function BlockRow({ block, selected, onSelect }: { block: PromptBlock; selected: boolean; onSelect: (id: string) => void }) {
  return (
    <ListRow
      onClick={() => onSelect(block.id)}
      selected={selected}
      title={block.title}
      meta={block.purpose}
      trailing={
        <>
          {block.overridden && <Badge tone="accent">수정됨</Badge>}
          <ChevronRight className="size-4 text-fg-muted" aria-hidden="true" />
        </>
      }
    />
  );
}

/** 단계별 블록 목록(개요 짜기 → 본문 쓰기 → 감수 → 용어 설명 → 질문 답변 → 구절 설명), 고급은 접어 둔다. */
export function PromptBlockList({ blocks, selectedId, onSelect }: PromptBlockListProps) {
  const basic = blocks.filter((b) => !b.advanced);
  const advanced = blocks.filter((b) => b.advanced);
  const stages = STAGE_ORDER.filter((stage) => basic.some((b) => b.stage === stage));
  return (
    <div className="flex flex-col gap-5">
      {stages.map((stage, index) => {
        const items = basic.filter((b) => b.stage === stage);
        return (
          <section key={stage} aria-labelledby={`stage-${stage}`}>
            <h2 id={`stage-${stage}`} className="text-xs font-semibold text-fg-secondary">
              {index + 1}. {items[0]?.stageTitle}
            </h2>
            <List>
              {items.map((b) => (
                <BlockRow key={b.id} block={b} selected={b.id === selectedId} onSelect={onSelect} />
              ))}
            </List>
          </section>
        );
      })}
      {advanced.length > 0 && (
        <details className="group border-t border-line pt-3" open={advanced.some((b) => b.id === selectedId || b.overridden)}>
          <summary className="flex min-h-touch cursor-pointer list-none items-center gap-1 text-xs font-semibold text-fg-secondary [&::-webkit-details-marker]:hidden">
            <ChevronRight className="size-4 transition-transform group-open:rotate-90" aria-hidden="true" />
            고급
          </summary>
          <List>
            {advanced.map((b) => (
              <BlockRow key={b.id} block={b} selected={b.id === selectedId} onSelect={onSelect} />
            ))}
          </List>
        </details>
      )}
    </div>
  );
}
