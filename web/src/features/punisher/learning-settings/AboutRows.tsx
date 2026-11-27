import { useAppStatus } from "@/src/shared/api/status";
import { isWitems } from "@/src/shared/config/runtime";
import { useT } from "@/src/shared/i18n";
import { BRAND_LINE, PunisherLogo } from "@/src/shared/ui/brand";
import { messages } from "./i18n";
import { SettingRow } from "./SettingRow";

/**
 * 설정 › 정보 구역: 버전(서버 상태에서) · 데이터 위치(단독판만 — 셸 안에서는 셸이 데이터를 맡는다) · 브랜드 한 줄과 forgen 링크.
 * 상태를 못 받았으면 버전·데이터 위치 행은 빼고 링크만 둔다(AI 연결 구역이 이미 불러오기 실패를 알린다).
 */
export function AboutRows() {
  const t = useT(messages);
  const status = useAppStatus();
  const data = status.data;
  const [before, after] = BRAND_LINE.split("forgen");
  return (
    <div className="flex flex-col">
      {data ? (
        <SettingRow label={t.aboutVersion}>
          <span className="text-sm text-fg-secondary">{data.appVersion}</span>
        </SettingRow>
      ) : null}
      {data && !isWitems() ? (
        <SettingRow label={t.aboutDataDir}>
          <span className="break-all font-mono text-xs text-fg-secondary">{data.dataDir}</span>
        </SettingRow>
      ) : null}
      <p className="flex items-center gap-2 py-3 text-sm text-fg-secondary">
        <PunisherLogo size={20} />
        <span>
          {before}
          <a href="https://github.com/forgen-team/forgen" target="_blank" rel="noreferrer" className="text-accent-text underline-offset-2 hover:underline">
            forgen
          </a>
          {after}
        </span>
      </p>
    </div>
  );
}
