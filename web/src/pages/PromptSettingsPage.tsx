import { PageHeader, useIsSplitLayout } from "@/src/shared/ui/kit";
import { BackLink } from "@/src/shared/ui/BackLink";
import { useSearchParams } from "@/src/shared/lib/router";
import { PromptSettingsWidget } from "@/src/widgets/punisher/prompt-settings";
import { useT } from "@/src/shared/i18n";
import { messages } from "./i18n";

/** 설정 > 강의 프롬프트 (witems: settings/learning/prompts). 한 단 배치의 블록 상세(?block=)에서 ← 는 목록으로. */
export function PromptSettingsPage() {
  const t = useT(messages);
  const params = useSearchParams();
  const split = useIsSplitLayout();
  const inDetail = !split && params.has("block");
  return (
    <div className="mx-auto flex w-full max-w-5xl flex-col xl:max-w-7xl">
      {inDetail ? <BackLink href="/settings/prompts" label={t.prompts} replace /> : <BackLink href="/settings" label={t.settings} />}
      <PageHeader title={t.prompts} className="pt-1" />
      <PromptSettingsWidget />
    </div>
  );
}
