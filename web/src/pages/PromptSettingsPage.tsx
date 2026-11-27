import { Page } from "@forgen-team/ui";
import { useIsSplitLayout } from "@/src/shared/lib/useIsSplitLayout";
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
    <div className="flex w-full flex-col">
      {inDetail ? <BackLink href="/settings/prompts" label={t.prompts} replace /> : <BackLink href="/settings" label={t.settings} />}
      {/* Witems 는 셸이 화면 제목을 그린다 — Page 는 셸 안에서 h1 을 그리지 않는다(같은 제목을 두 번 그리지 않는다). */}
      <Page title={t.prompts} width="full">
        <PromptSettingsWidget />
      </Page>
    </div>
  );
}
