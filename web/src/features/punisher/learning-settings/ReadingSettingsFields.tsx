import { setUiPrefs, useUiPrefs, type FontSize, type LineWidth, type UiPrefs } from "@/src/shared/lib/ui-prefs";
import { useT } from "@/src/shared/i18n";
import { studyMessages } from "./i18n";
import { SelectRow, SwitchRow } from "./SettingRow";
import { useSavedFlash } from "./useFieldSaves";

/**
 * 설정 › 읽기 구역(이 기기만): 강의 글자 크기 · 줄 폭 · 복습 단축키. 바꾸면 바로 저장(localStorage)된다.
 * 서버 설정이 아니라 실패하지 않으므로 오류 칸은 없다.
 */
export function ReadingSettingsFields() {
  const t = useT(studyMessages);
  const ui = useUiPrefs();
  const { savedKey, flash } = useSavedFlash();
  const change = (patch: Partial<UiPrefs>) => {
    setUiPrefs(patch);
    flash(Object.keys(patch)[0] ?? "");
  };
  return (
    <div className="flex flex-col">
      <SelectRow
        label={t.fontSize}
        saved={savedKey === "fontSize"}
        value={ui.fontSize}
        onChange={(v) => change({ fontSize: v as FontSize })}
        options={[
          { value: "sm", label: t.fontSm },
          { value: "md", label: t.fontMd },
          { value: "lg", label: t.fontLg },
        ]}
      />
      <SelectRow
        label={t.lineWidth}
        saved={savedKey === "lineWidth"}
        value={ui.lineWidth}
        onChange={(v) => change({ lineWidth: v as LineWidth })}
        options={[
          { value: "narrow", label: t.widthNarrow },
          { value: "normal", label: t.widthNormal },
          { value: "wide", label: t.widthWide },
        ]}
      />
      <SwitchRow
        label={t.shortcuts}
        hint={t.shortcutsHint}
        saved={savedKey === "shortcuts"}
        checked={ui.shortcuts}
        onChange={(checked) => change({ shortcuts: checked })}
      />
    </div>
  );
}
