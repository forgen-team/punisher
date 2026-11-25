import { useState, type MouseEvent } from "react";
import { Download, ScrollText } from "lucide-react";
import { List, ListRow, PageHeader, Section, ThemeSelect, notify } from "@/src/shared/ui/kit";
import { isWitems } from "@/src/shared/config/runtime";
import { Link } from "@/src/shared/lib/router";
import { LearningSettingsForm, ReadingSettingsFields, StudySettingsFields } from "@/src/features/punisher/learning-settings";
import { punisherApiBase } from "@/src/shared/api/punisher";
import { BRAND_LINE, PunisherLogo } from "@/src/shared/ui/brand";
import { useT } from "@/src/shared/i18n";
import { messages } from "./i18n";
import { AiConnectionPanel } from "@/src/widgets/claude-status";

/** Content-Disposition 의 파일 이름(없으면 기본). */
function exportFileName(header: string | null): string {
  const match = header?.match(/filename\*?=(?:UTF-8'')?"?([^";]+)"?/i);
  return match?.[1] ? decodeURIComponent(match[1]) : "punisher-export.zip";
}

/**
 * 데이터 내보내기: 링크 그대로(새 탭·다른 이름으로 저장 가능) 두되, 보통 클릭은 받아서 저장한다 —
 * 실패하면 브라우저가 서버 오류 JSON 을 그대로 여는 대신 앱 안 알림으로 알린다.
 */
function ExportRow() {
  const t = useT(messages);
  const [busy, setBusy] = useState(false);
  const href = `${punisherApiBase()}/export`;
  async function download(event: MouseEvent) {
    if (event.metaKey || event.ctrlKey || event.shiftKey || event.button !== 0) { return; }
    event.preventDefault();
    if (busy) { return; }
    setBusy(true);
    try {
      const res = await fetch(href, { credentials: "same-origin" });
      if (!res.ok) { throw new Error(`export ${res.status}`); }
      const url = URL.createObjectURL(await res.blob());
      const a = document.createElement("a");
      a.href = url;
      a.download = exportFileName(res.headers.get("Content-Disposition"));
      a.click();
      setTimeout(() => URL.revokeObjectURL(url), 1000);
    } catch (err) {
      console.warn("punisher: export failed", err);
      notify(t.exportFailed, { variant: "error" });
    } finally {
      setBusy(false);
    }
  }
  return <ListRow href={href} onClick={download} leading={<Download />} title={t.export} meta={busy ? t.exporting : t.exportMeta} />;
}

/**
 * 설정: 언어 · 예시 코드 언어 · 화면 테마 · AI 연결(제공자 선택·상태) · 데이터 위치.
 * Witems: 제목은 셸이 그리고(데스크톱 h1·폰 상단 바), 테마는 셸 것을 따르므로 화면 구역을 두지 않는다.
 */
export function SettingsPage() {
  const t = useT(messages);
  const inWitems = isWitems();
  return (
    <div className="mx-auto flex w-full max-w-2xl flex-col gap-6">
      {inWitems ? null : <PageHeader title={t.settings} />}
      <Section title={t.sectionLearning}>
        <LearningSettingsForm />
      </Section>
      <Section title={t.sectionReview}>
        <StudySettingsFields />
      </Section>
      <Section title={t.sectionReading}>
        <ReadingSettingsFields />
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
          <ExportRow />
        </List>
      </Section>
      {inWitems ? null : (
        <Section title={t.sectionScreen}>
          <ThemeSelect className="w-fit" />
        </Section>
      )}
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
