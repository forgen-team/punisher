import { PageHeader } from "@/src/shared/ui/kit";
import { BackLink } from "@/src/shared/ui/BackLink";
import { PromptSettingsWidget } from "@/src/widgets/punisher/prompt-settings";

/** 설정 > 강의 프롬프트 (witems: settings/learning/prompts). */
export function PromptSettingsPage() {
  return (
    <div className="mx-auto flex w-full max-w-5xl flex-col">
      <BackLink href="/settings" label="설정" />
      <PageHeader title="강의 프롬프트" className="pt-1" />
      <PromptSettingsWidget />
    </div>
  );
}
