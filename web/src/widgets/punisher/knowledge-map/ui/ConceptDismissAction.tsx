import { Button } from "@/src/shared/ui/kit";
import { useConceptDismiss } from "@/src/entities/punisher";
import { InlineError } from "@/src/shared/ui/punisher";
import { useT } from "@/src/shared/i18n";
import { messages } from "../i18n";

interface ConceptDismissActionProps {
  conceptId: string;
  dismissed: boolean;
  /** 완료·이미 앎 개념은 이미 추천에서 빠지므로 '필요 없음'을 내놓지 않는다(뺀 상태면 되돌리기는 보인다). */
  learned: boolean;
}

/** 개념 상세의 '필요 없음' 동작: 빼기, 또는 "필요 없음으로 뺐어요 · 되돌리기". */
export function ConceptDismissAction({ conceptId, dismissed, learned }: ConceptDismissActionProps) {
  const t = useT(messages);
  const toggle = useConceptDismiss();
  const run = (next: boolean) => toggle.mutate({ conceptId, dismissed: next });

  if (dismissed) {
    return (
      // 버튼 줄 위에 한 줄을 통째로 쓴다(좁은 폭에서 '·'만 남고 줄이 끊기던 문제).
      <div className="order-first flex basis-full items-center justify-between gap-2 text-sm text-fg-secondary" aria-live="polite">
        <span className="min-w-0">{t.dismissedNotice}</span>
        <Button variant="ghost" className="shrink-0" loading={toggle.isPending} onClick={() => run(false)}>
          {t.dismissUndo}
        </Button>
        <InlineError error={toggle.error} />
      </div>
    );
  }
  if (learned) { return null; }
  return (
    <>
      <Button variant="ghost" loading={toggle.isPending} onClick={() => run(true)}>
        {t.dismiss}
      </Button>
      <InlineError error={toggle.error} />
    </>
  );
}
