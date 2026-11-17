
import { X } from "lucide-react";
import { Button, IconButton } from "@/src/shared/ui/kit";
import { useT } from "@/src/shared/i18n";
import { messages } from "./i18n";

interface KnownOnboardingPromptProps {
  onStart: () => void;
  onDismiss: () => void;
}

/** 첫 진입 한 줄 안내(카드 아님): 지도 위 구분선 행. */
export function KnownOnboardingPrompt({ onStart, onDismiss }: KnownOnboardingPromptProps) {
  const t = useT(messages);
  return (
    <div role="region" aria-label={t.promptRegion} className="flex items-center gap-2 border-y border-line py-1.5">
      <p className="min-w-0 flex-1 text-sm text-fg">{t.promptText}</p>
      <Button size="sm" variant="secondary" onClick={onStart}>
        {t.promptStart}
      </Button>
      <IconButton aria-label={t.promptClose} size="sm" onClick={onDismiss}>
        <X aria-hidden="true" />
      </IconButton>
    </div>
  );
}
