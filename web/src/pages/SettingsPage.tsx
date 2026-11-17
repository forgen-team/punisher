import { CheckCircle2, AlertTriangle, Download, ScrollText } from "lucide-react";
import { List, ListRow, PageHeader, Section, Skeleton, ThemeSelect } from "@/src/shared/ui/kit";
import { Link } from "@/src/shared/lib/router";
import { LearningSettingsForm } from "@/src/features/punisher/learning-settings";
import { useAppStatus } from "@/src/shared/api/status";
import { PUNISHER_API_BASE } from "@/src/shared/api/punisher";
import { BRAND_LINE, PunisherLogo } from "@/src/shared/ui/brand";
import { useT } from "@/src/shared/i18n";
import { messages } from "./i18n";
import { messages as statusMessages, statusText } from "@/src/widgets/claude-status";

/** 설정: 언어 · 예시 코드 언어 · 화면 테마 · Claude Code 연결 상태 · 데이터 위치. */
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
            href={`${PUNISHER_API_BASE}/export`}
            leading={<Download />}
            title={t.export}
            meta={t.exportMeta}
          />
        </List>
      </Section>
      <Section title={t.sectionScreen}>
        <ThemeSelect className="w-fit" />
      </Section>
      <Section title={t.sectionClaude}>
        <ClaudeStatusPanel />
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

function ClaudeStatusPanel() {
  const t = useT(messages);
  const ts = useT(statusMessages);
  const status = useAppStatus();
  if (status.isLoading) { return <Skeleton variant="block" className="h-20" />; }
  if (!status.data) { return <p className="text-sm text-danger">{t.statusLoadFailed}</p>; }
  const { claude, dataDir, appVersion } = status.data;
  const { message } = statusText(ts, claude.code, status.data.message);
  const ok = !claude.code;

  return (
    <div className="flex flex-col gap-3 text-sm">
      <p className="flex items-center gap-2">
        {ok ? (
          <CheckCircle2 className="h-4 w-4 text-success" aria-hidden="true" />
        ) : (
          <AlertTriangle className="h-4 w-4 text-warning" aria-hidden="true" />
        )}
        <span className="font-medium text-fg">
          {ok ? t.connected : message}
        </span>
      </p>
      <dl className="grid grid-cols-[auto_1fr] gap-x-4 gap-y-1 text-fg-secondary">
        <dt>Claude Code</dt>
        <dd className="text-fg">{claude.installed ? claude.version || t.installed : t.notInstalled}</dd>
        <dt>{t.login}</dt>
        <dd className="text-fg">{claude.loggedIn === null ? t.loginUnknown : claude.loggedIn ? t.loggedIn : t.loggedOut}</dd>
        <dt>{t.activeJobs}</dt>
        <dd className="text-fg">{t.activeCount(claude.active, claude.queued)}</dd>
        <dt>{t.dataDir}</dt>
        <dd className="break-all font-mono text-xs text-fg">{dataDir}</dd>
        <dt>{t.version}</dt>
        <dd className="text-fg">{appVersion}</dd>
      </dl>
    </div>
  );
}
