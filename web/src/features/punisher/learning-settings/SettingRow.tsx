import type { ReactNode } from "react";
import { Select, Switch, type SelectOption } from "@forgen-team/ui";
import { useT } from "@/src/shared/i18n";
import { messages } from "./i18n";

/**
 * 설정 행의 오른쪽 조작 폭 — 폰에서는 좁게. 부품의 이름 칸은 sr-only 라 높이가 0 인데도 위아래 간격(gap)은 남으므로 없앤다.
 * export: 글자 칸처럼 폭만 다른 조작이 같은 간격 처리를 받는다.
 */
export const CONTROL_NO_LABEL_GAP = "[&_.seed-field__root]:gap-0";
const CONTROL_WIDTH = `w-44 sm:w-56 ${CONTROL_NO_LABEL_GAP}`;

interface SettingRowProps {
  label: string;
  /** 이름 아래 한 줄 설명(xs, 흐린 글씨) */
  hint?: ReactNode;
  /** 방금 바로 저장했으면 조작 왼쪽에 "저장했어요" 를 잠깐 보인다 */
  saved?: boolean;
  /** 이 칸의 저장 오류 — 조작 아래 */
  error?: string;
  children: ReactNode;
}

/**
 * 설정 한 행(시안 .set-row): 왼쪽 이름(+힌트), 오른쪽 조작. 아래 가는 선으로 이어진다.
 * 조작 부품의 이름은 화면에 두 번 보이지 않게 sr-only 로 달고, 눈에 보이는 이름은 이 행이 그린다.
 */
export function SettingRow({ label, hint, saved, error, children }: SettingRowProps) {
  const t = useT(messages);
  return (
    <div className="flex min-h-[52px] flex-wrap items-center justify-between gap-x-4 gap-y-2 border-b border-line-subtle py-2">
      <div className="flex min-w-0 flex-1 basis-40 flex-col">
        <span className="text-sm text-fg" aria-hidden="true">{label}</span>
        {hint ? <span className="text-xs text-fg-muted">{hint}</span> : null}
      </div>
      <div className="flex min-w-0 flex-col items-end gap-1">
        <div className="flex items-center gap-2">
          <span aria-live="polite" className="text-xs text-success">{saved ? t.saved : ""}</span>
          {children}
        </div>
        {error ? <p role="alert" className="text-xs text-danger">{error}</p> : null}
      </div>
    </div>
  );
}

interface SelectRowProps extends Omit<SettingRowProps, "children"> {
  value: string;
  options: SelectOption[];
  onChange: (value: string) => void;
  placeholder?: string;
}

/** 이름 + 고르기(바로 저장). */
export function SelectRow({ value, options, onChange, placeholder, ...row }: SelectRowProps) {
  return (
    <SettingRow {...row}>
      <Select
        label={<span className="sr-only">{row.label}</span>}
        className={CONTROL_WIDTH}
        invalid={Boolean(row.error)}
        value={value}
        placeholder={placeholder}
        options={options}
        onValueChange={onChange}
      />
    </SettingRow>
  );
}

interface SwitchRowProps extends Omit<SettingRowProps, "children"> {
  checked: boolean;
  onChange: (checked: boolean) => void;
}

/** 이름 + 스위치(바로 저장). */
export function SwitchRow({ checked, onChange, ...row }: SwitchRowProps) {
  return (
    <SettingRow {...row}>
      <Switch checked={checked} onCheckedChange={onChange} inputProps={{ "aria-label": row.label }} />
    </SettingRow>
  );
}
