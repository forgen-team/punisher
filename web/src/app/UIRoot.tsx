import type { ReactNode } from "react";
import { FeedbackProvider, UIProvider } from "@forgen-team/ui";
import { useT } from "@/src/shared/i18n";
import { Link } from "@/src/shared/lib/router";
import { uiMessages } from "./i18n";

/**
 * 부품 키트(@forgen-team/ui) 뿌리: 링크는 앱 라우터 Link, 오버레이·토스트는 portalRoot(Witems 앱 모드는 ctx.portal),
 * 셸 안(inShell)이면 h1·← 는 셸이 그린다. useConfirm·useToast 는 FeedbackProvider 아래에서만 쓴다.
 */
export function UIRoot({ inShell = false, portal = null, children }: { inShell?: boolean; portal?: HTMLElement | null; children: ReactNode }) {
  const messages = useT(uiMessages);
  return (
    <UIProvider linkAs={Link} inShell={inShell} portalRoot={portal} messages={messages}>
      <FeedbackProvider>{children}</FeedbackProvider>
    </UIProvider>
  );
}
