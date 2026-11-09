import { ChevronLeft } from "lucide-react";
import { Link } from "@/src/shared/lib/router";
import { PageHeader, cn, focusRing } from "@/src/shared/ui/kit";
import { PromptSettingsWidget } from "@/src/widgets/punisher/prompt-settings";

/** 설정 > 강의 프롬프트 (witems: settings/learning/prompts). */
export function PromptSettingsPage() {
  return (
    <div className="mx-auto flex w-full max-w-5xl flex-col pb-10">
      <Link
        href="/settings"
        className={cn("-ml-2 inline-flex min-h-touch w-fit items-center gap-1 rounded-md px-2 text-sm text-fg-secondary hover:text-fg", focusRing)}
      >
        <ChevronLeft className="h-4 w-4" aria-hidden="true" />
        설정
      </Link>
      <PageHeader title="강의 프롬프트" className="pt-1" />
      <PromptSettingsWidget />
    </div>
  );
}
