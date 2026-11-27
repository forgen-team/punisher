import { useState, type MouseEvent, type ReactNode } from "react";
import { useAtom } from "jotai";
import { Download, ScrollText } from "lucide-react";
import { List, ListRow, Page, Section, SegmentedControl, SegmentedControlItem, useToast } from "@forgen-team/ui";
import { isWitems, standaloneColumnClass } from "@/src/shared/config/runtime";
import { themePreferenceAtom, type ThemePreference } from "@/src/shared/lib/theme";
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
 * ui ListRow 의 링크는 onClick 을 받지 않으므로 감싼 곳에서 클릭을 먼저 받는다(capture).
 */
function ExportRows({ children }: { children: ReactNode }) {
  const t = useT(messages);
  const toast = useToast();
  const [busy, setBusy] = useState(false);
  const href = `${punisherApiBase()}/export`;
  async function download(event: MouseEvent) {
    if (event.metaKey || event.ctrlKey || event.shiftKey || event.button !== 0) { return; }
    const target = event.target instanceof Element ? event.target.closest("a") : null;
    if (!target || target.getAttribute("href") !== href) { return; }
    event.preventDefault();
    event.stopPropagation();
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
      toast.show(t.exportFailed, { tone: "danger" });
    } finally {
      setBusy(false);
    }
  }
  return (
    <div onClickCapture={(event) => { void download(event); }}>
      <List>
        {children}
        <ListRow href={href} leading={<Download />} title={t.export} meta={busy ? t.exporting : t.exportMeta} />
      </List>
    </div>
  );
}

const THEMES: ThemePreference[] = ["light", "dark", "system"];

/** 화면 테마(라이트·다크·시스템) — 하나만 고르므로 SegmentedControl. */
function ThemeChoice() {
  const t = useT(messages);
  const [preference, setPreference] = useAtom(themePreferenceAtom);
  const label = { light: t.themeLight, dark: t.themeDark, system: t.themeSystem } as const;
  return (
    <SegmentedControl aria-label={t.theme} value={preference} onValueChange={(value) => setPreference(value as ThemePreference)}>
      {THEMES.map((value) => (
        <SegmentedControlItem key={value} value={value}>{label[value]}</SegmentedControlItem>
      ))}
    </SegmentedControl>
  );
}

/**
 * 설정: 언어 · 예시 코드 언어 · 화면 테마 · AI 연결(제공자 선택·상태) · 데이터 위치.
 * Witems: 제목은 셸이 그리고(데스크톱 h1·폰 상단 바 — Page 가 셸 안에서는 h1 을 그리지 않는다), 테마는 셸 것을 따르므로 화면 구역을 두지 않는다.
 */
export function SettingsPage() {
  const t = useT(messages);
  const inWitems = isWitems();
  return (
    <Page title={t.settings} width="full" className={standaloneColumnClass()}>
    <div className="flex w-full flex-col gap-6">
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
        <ExportRows>
          <ListRow
            href="/settings/prompts"
            leading={<ScrollText />}
            title={t.prompts}
            meta={t.promptsMeta}
          />
        </ExportRows>
      </Section>
      {inWitems ? null : (
        <Section title={t.sectionScreen}>
          <ThemeChoice />
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
    </Page>
  );
}
