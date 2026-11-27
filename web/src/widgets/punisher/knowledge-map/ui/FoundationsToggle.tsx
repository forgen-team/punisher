import { Button } from "@forgen-team/ui";
import { Link } from "@/src/shared/lib/router";
import { useT } from "@/src/shared/i18n";
import { messages } from "../i18n";

interface FoundationsToggleProps {
  /** 기본 숨김인 0단계 기초 개념 수 */
  count: number;
  expanded: boolean;
  onToggle: () => void;
}

/**
 * "기초 개념 n개 숨김(설명 수준 설정) · 보이기" 한 줄 — 왜 숨었는지(설명 수준이 실무자·숙련자 또는 미설정)를 보이고,
 * 이유는 설정으로 가는 링크, 보이기는 이 지도에서만 펼친다(P4·점검 28).
 */
export function FoundationsToggle({ count, expanded, onToggle }: FoundationsToggleProps) {
  const t = useT(messages);
  return (
    <div className="flex min-h-touch items-center justify-between gap-3 rounded-lg border border-dashed border-line px-3 py-1.5">
      <p className="min-w-0 text-sm text-fg-secondary">
        <span className="font-medium text-fg">{t.foundationsTitle}</span>{" "}
        {expanded ? t.foundationsShown(count) : t.foundationsHidden(count)}
        {expanded ? null : (
          <>
            {" ("}
            <Link href="/settings" className="underline underline-offset-2 hover:text-fg">{t.foundationsReason}</Link>
            {")"}
          </>
        )}
      </p>
      <Button variant="ghost" size="sm" aria-expanded={expanded} onClick={onToggle}>
        {expanded ? t.foundationsCollapse : t.foundationsExpand}
      </Button>
    </div>
  );
}
