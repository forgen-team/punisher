import { AlertTriangle, CheckCircle2, RefreshCw } from "lucide-react";
import { useQueryClient } from "@tanstack/react-query";
import { describeSettingsError, useSaveLearningSettings } from "@/src/entities/punisher";
import {
  AI_PROVIDER_NAMES,
  APP_STATUS_KEY,
  providerStatuses,
  selectedProviderId,
  useAppStatus,
  type AiProvider,
  type AppStatus,
  type ProviderStatus,
} from "@/src/shared/api/status";
import { Button, Skeleton } from "@forgen-team/ui";
import { cn } from "@/src/shared/lib/cn";
import { isWitems } from "@/src/shared/config/runtime";
import { SettingRow } from "@/src/features/punisher/learning-settings";
import { useT } from "@/src/shared/i18n";
import { messages, panelMessages, statusText, witemsStatusText } from "./i18n";

/**
 * 설정 › AI 연결: 고른 AI 하나가 강의 생성·검수·질문 답·채점을 모두 맡는다.
 * 설치됐고 로그인 안 됨이 확실하지 않은 제공자만 고를 수 있다(agy 는 로그인을 미리 알 수 없어 고를 수 있다).
 * 바꾸면 PUT /me/settings { aiProvider } 후 상태를 다시 받는다(서버는 다음 호출부터 바로 쓴다) — 고르는 즉시 저장이라 저장 버튼이 없다.
 * 셸 안: 연결 상태 · 진행 중 작업 수 / 단독판: 제공자 고르기 · 다시 확인 · 작업 수.
 * (CLI 계정 연결이라 API 키 칸은 없다. 버전·데이터 위치는 설정 › 정보 구역.)
 */
export function AiConnectionPanel() {
  const t = useT(panelMessages);
  const ts = useT(messages);
  const status = useAppStatus();
  const save = useSaveLearningSettings();
  const queryClient = useQueryClient();
  if (status.isLoading) { return <Skeleton lines={3} className="h-20" />; }
  if (!status.data) { return <p className="text-sm text-danger">{t.statusLoadFailed}</p>; }

  const data = status.data;
  const providers = providerStatuses(data);
  const serverSelected = selectedProviderId(data);
  const selected: AiProvider = (save.isPending && save.variables?.aiProvider) || serverSelected;
  const current = providers.find((p) => p.id === serverSelected);
  const { message } = statusText(ts, current?.code, data.message);
  const ok = current !== undefined && !current.code;
  const { claude } = data;
  if (isWitems()) {
    // Witems: AI 는 셸의 연결 하나 — 제공자 고르기·로컬 CLI·데이터 위치는 의미가 없다.
    return (
      <div className="flex flex-col gap-2 text-sm">
        <p className="flex items-center gap-2">
          {ok ? <CheckCircle2 className="h-4 w-4 shrink-0 text-success" aria-hidden="true" /> : <AlertTriangle className="h-4 w-4 shrink-0 text-warning" aria-hidden="true" />}
          <span className="font-medium text-fg">{ok ? t.witemsConnected : witemsStatusText(ts, current?.code ?? "claude_unavailable").message}</span>
        </p>
        <ActiveJobsRow label={t.activeJobs} value={t.activeCount(claude.active, claude.queued)} />
      </div>
    );
  }

  const choose = (id: AiProvider) => {
    if (id === serverSelected || save.isPending) { return; }
    save.mutate(
      { aiProvider: id },
      {
        // 저장이 끝나고 상태를 다시 받기 전에 예전 제공자로 잠깐 돌아가 보이지 않게, 캐시의 고른 AI 를 먼저 바꾼다.
        onSuccess: () => queryClient.setQueryData<AppStatus>(APP_STATUS_KEY, (prev) => (prev ? { ...prev, provider: id } : prev)),
        onSettled: () => void status.refetch(),
      },
    );
  };

  return (
    <div className="flex flex-col gap-3 text-sm">
      <p className="flex items-center gap-2">
        {ok ? (
          <CheckCircle2 className="h-4 w-4 shrink-0 text-success" aria-hidden="true" />
        ) : (
          <AlertTriangle className="h-4 w-4 shrink-0 text-warning" aria-hidden="true" />
        )}
        <span className="font-medium text-fg">{ok ? t.connected(AI_PROVIDER_NAMES[serverSelected]) : message}</span>
      </p>
      <fieldset className="flex flex-col gap-1" disabled={save.isPending}>
        <legend className="mb-1 text-fg-secondary">{t.legend}</legend>
        {providers.map((p) => (
          <ProviderOption
            key={p.id}
            provider={p}
            checked={p.id === selected}
            onChoose={choose}
            statusLine={providerStatusLine(t, p)}
            selectedLabel={t.selected}
          />
        ))}
      </fieldset>
      {save.error && <p className="text-sm text-danger" role="alert">{describeSettingsError(save.error)}</p>}
      <div>
        <Button
          variant="ghost"
          size="sm"
          className="min-h-11"
          onClick={() => void status.refetch()}
          disabled={status.isFetching}
        >
          <RefreshCw className={status.isFetching ? "h-4 w-4 animate-spin" : "h-4 w-4"} aria-hidden="true" />
          {ts.recheck}
        </Button>
      </div>
      <ActiveJobsRow label={t.activeJobs} value={t.activeCount(claude.active, claude.queued)} />
    </div>
  );
}

/** 진행 중인 생성 수 한 행(설정 행 모양). 버전·데이터 위치는 설정 › 정보 구역이 보여 준다. */
function ActiveJobsRow({ label, value }: { label: string; value: string }) {
  return (
    <SettingRow label={label}>
      <span className="text-sm text-fg-secondary">{value}</span>
    </SettingRow>
  );
}

/** 설치되어 있고 로그인 안 됨이 확인되지 않았으면 고를 수 있다. */
function selectable(p: ProviderStatus): boolean {
  return p.installed && p.loggedIn !== false;
}

/** "설치됨 · 1.2.3 · 로그인됨" / "설치 안 됨" */
function providerStatusLine(t: (typeof panelMessages)["ko"], p: ProviderStatus): string {
  if (!p.installed) { return t.notInstalled; }
  const login = p.loggedIn === null ? t.loginUnknown : p.loggedIn ? t.loggedIn : t.loggedOut;
  return [t.installed, p.version, login].filter(Boolean).join(" · ");
}

function ProviderOption({
  provider,
  checked,
  onChoose,
  statusLine,
  selectedLabel,
}: {
  provider: ProviderStatus;
  checked: boolean;
  onChoose: (id: AiProvider) => void;
  statusLine: string;
  selectedLabel: string;
}) {
  const disabled = !selectable(provider) && !checked;
  const name = AI_PROVIDER_NAMES[provider.id];
  return (
    <label
      className={cn(
        "flex min-h-11 items-center gap-3 border px-3 py-2",
        checked ? "border-accent bg-accent-subtle" : "border-line",
        disabled ? "cursor-not-allowed opacity-60" : "cursor-pointer",
      )}
    >
      <input
        type="radio"
        name="ai-provider"
        value={provider.id}
        checked={checked}
        disabled={disabled}
        onChange={() => onChoose(provider.id)}
        className="h-4 w-4 shrink-0 accent-[var(--color-accent)]"
      />
      <span className="flex min-w-0 flex-1 flex-col">
        <span className="font-medium text-fg">{name}</span>
        <span className="text-xs text-fg-secondary">{statusLine}</span>
      </span>
      {checked && <span className="shrink-0 text-xs font-medium text-accent-text">{selectedLabel}</span>}
    </label>
  );
}
