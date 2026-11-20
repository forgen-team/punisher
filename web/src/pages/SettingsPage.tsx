import { Download, ScrollText } from "lucide-react";
import { List, ListRow, PageHeader, Section, ThemeSelect } from "@/src/shared/ui/kit";
import { Link } from "@/src/shared/lib/router";
import { LearningSettingsForm } from "@/src/features/punisher/learning-settings";
import { punisherApiBase } from "@/src/shared/api/punisher";
import { BRAND_LINE, PunisherLogo } from "@/src/shared/ui/brand";
import { useT } from "@/src/shared/i18n";
import { messages } from "./i18n";
import { AiConnectionPanel } from "@/src/widgets/claude-status";

/** 설정: 언어 · 예시 코드 언어 · 화면 테마 · AI 연결(제공자 선택·상태) · 데이터 위치. */
export function SettingsPage() {
  const t = useT(messages);
  return (
    <div className="mx-auto flex w-full max-w-2xl flex-col gap-6">
      <PageHeader title={t.settings} />
      <Section title={t.sectionLearning}>
        <LearningSettingsForm />
      </Section>
      <Section title={t.sectionAdvanced}>
        <List>
          <ListRow
            href="/settings/prompts"
            linkAs={Link}
            leading={<ScrollText />}
            title={t.prompts}
            meta={t.promptsMeta}
          />
          <ListRow
            href={`${punisherApiBase()}/export`}
            leading={<Download />}
            title={t.export}
            meta={t.exportMeta}
          />
        </List>
      </Section>
      <Section title={t.sectionScreen}>
        <ThemeSelect className="w-fit" />
      </Section>
      <Section title={t.sectionAi}>
        <AiConnectionPanel />
      </Section>
      <Section title={t.sectionAbout}>
        <p className="flex items-center gap-2 text-sm text-fg-secondary">
          <PunisherLogo size={20} />
          <span>
            {BRAND_LINE.split("forgen")[0]}
            <a href="https://github.com/forgen-team/forgen" target="_blank" rel="noreferrer" className="text-accent-text underline-offset-2 hover:underline">
              forgen
            </a>
            {BRAND_LINE.split("forgen")[1]}
          </span>
        </p>
      </Section>
    </div>
  );
}
