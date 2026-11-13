import { PageHeader, useIsSplitLayout } from "@/src/shared/ui/kit";
import { BackLink } from "@/src/shared/ui/BackLink";
import { useSearchParams } from "@/src/shared/lib/router";
import { PromptSettingsWidget } from "@/src/widgets/punisher/prompt-settings";

/** 설정 > 강의 프롬프트 (witems: settings/learning/prompts). 한 단 배치의 블록 상세(?block=)에서 ← 는 목록으로. */
export function PromptSettingsPage() {
  const params = useSearchParams();
  const split = useIsSplitLayout();
  const inDetail = !split && params.has("block");
  return (
    <div className="mx-auto flex w-full max-w-5xl flex-col xl:max-w-7xl">
      {inDetail ? <BackLink href="/settings/prompts" label="강의 프롬프트" replace /> : <BackLink href="/settings" label="설정" />}
      <PageHeader title="강의 프롬프트" className="pt-1" />
      <PromptSettingsWidget />
    </div>
  );
}
