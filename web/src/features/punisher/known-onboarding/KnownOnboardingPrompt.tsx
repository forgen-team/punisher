
import { X } from "lucide-react";
import { Button, IconButton } from "@/src/shared/ui/kit";

interface KnownOnboardingPromptProps {
  onStart: () => void;
  onDismiss: () => void;
}

/** 첫 진입 한 줄 안내(카드 아님): 지도 위 구분선 행. */
export function KnownOnboardingPrompt({ onStart, onDismiss }: KnownOnboardingPromptProps) {
  return (
    <div role="region" aria-label="지도 시작 안내" className="flex items-center gap-2 border-y border-line py-1.5">
      <p className="min-w-0 flex-1 text-sm text-fg">이미 아는 영역을 표시하면 지도가 열려요</p>
      <Button size="sm" variant="secondary" onClick={onStart}>
        표시하기
      </Button>
      <IconButton aria-label="안내 닫기" size="sm" onClick={onDismiss}>
        <X aria-hidden="true" />
      </IconButton>
    </div>
  );
}
