import { Button } from "@/src/shared/ui/kit";
import { useT } from "@/src/shared/i18n";
import { messages } from "../i18n";

interface FoundationsToggleProps {
  /** 기본 숨김인 0단계 기초 개념 수 */
  count: number;
  expanded: boolean;
  onToggle: () => void;
}

/** "0단계 기초 · 숨김 · 펼치기" 한 줄 — 실무자·숙련자(또는 미설정)에게 접혀 있는 기초 트랙을 지도에 펼친다. */
export function FoundationsToggle({ count, expanded, onToggle }: FoundationsToggleProps) {
  const t = useT(messages);
  return (
    <div className="flex min-h-touch items-center justify-between gap-3 rounded-lg border border-dashed border-line px-3 py-1.5">
      <p className="min-w-0 text-sm text-fg-secondary">
        <span className="font-medium text-fg">{t.foundationsTitle}</span>
        {" · "}
        {expanded ? t.foundationsShown(count) : t.foundationsHidden(count)}
      </p>
      <Button variant="ghost" size="sm" aria-expanded={expanded} onClick={onToggle}>
        {expanded ? t.foundationsCollapse : t.foundationsExpand}
      </Button>
    </div>
  );
}
