
import { BackLink } from "@/src/shared/ui/BackLink";
import { useT } from "@/src/shared/i18n";
import { messages } from "../i18n";

/** '← 학습' — 단독판은 셸 상단 바에 ← 가 없어 BackLink(모든 폭)로 둔다(witems 는 lg 이상 DesktopBackLink). */
export function BackToLearnLink({ className }: { className?: string }) {
  const t = useT(messages);
  return <BackLink href="/" label={t.backToLearn} className={className} />;
}
