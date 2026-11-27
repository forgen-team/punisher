import { useState, type MouseEvent, type ReactNode } from "react";
import { useAtom } from "jotai";
import { Download, ScrollText } from "lucide-react";
import { List, ListRow, Section, SegmentedControl, SegmentedControlItem, useToast } from "@forgen-team/ui";
import { isWitems } from "@/src/shared/config/runtime";
import { themePreferenceAtom, type ThemePreference } from "@/src/shared/lib/theme";
import {
  AboutRows,
  LearningSettingsForm,
  ReadingSettingsFields,
  ReviewSettingsFields,
  SettingRow,
  learningSettingsMessages,
  useSavedFlash,
} from "@/src/features/punisher/learning-settings";
import { punisherApiBase } from "@/src/shared/api/punisher";
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

/** 화면 테마(라이트·다크·시스템) — 하나만 고르므로 SegmentedControl. 고르면 바로 저장(기기에). */
function ThemeChoice() {
  const t = useT(messages);
  const [preference, setPreference] = useAtom(themePreferenceAtom);
  const { savedKey, flash } = useSavedFlash();
  const label = { light: t.themeLight, dark: t.themeDark, system: t.themeSystem } as const;
  return (
    <SettingRow label={t.theme} saved={savedKey === "theme"}>
      <SegmentedControl
        aria-label={t.theme}
        value={preference}
        onValueChange={(value) => { setPreference(value as ThemePreference); flash("theme"); }}
      >
        {THEMES.map((value) => (
          <SegmentedControlItem key={value} value={value}>{label[value]}</SegmentedControlItem>
        ))}
      </SegmentedControl>
    </SettingRow>
  );
}

/**
 * 설정(`/settings`, T-settings 단일): 학습 · 복습 · 읽기 · AI 연결 · 고급 · 화면(단독판만) · 정보 — 셸 안 6구역, 단독판 7구역.
 * 제목("punisher 설정")·←·폭(narrow)은 틀이 공통 Page 로 그리므로 이 화면은 h1·←·폭을 두지 않는다.
 * 저장은 한 구역 안에서 한 가지 방식만 쓴다 — 고르기·스위치·칩은 바로 저장(그 자리에 "저장했어요"), 숫자 칸은 칸을 떠날 때 저장.
 * 테마는 단독판에서만 고른다(Witems 는 셸 테마를 따른다).
 */
export function SettingsPage() {
  const t = useT(messages);
  const tl = useT(learningSettingsMessages);
  return (
    <div className="flex w-full flex-col gap-8">
      <Section title={t.sectionLearning}>
        <LearningSettingsForm />
      </Section>
      <Section title={tl.sectionReview}>
        <ReviewSettingsFields />
      </Section>
      <Section title={t.sectionReading} description={tl.deviceHint}>
        <ReadingSettingsFields />
      </Section>
      <Section title={t.sectionAi}>
        <AiConnectionPanel />
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
      {isWitems() ? null : (
        <Section title={t.sectionScreen}>
          <ThemeChoice />
        </Section>
      )}
      <Section title={t.sectionAbout}>
        <AboutRows />
      </Section>
    </div>
  );
}
