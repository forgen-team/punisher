import { CheckCircle2, AlertTriangle, Download, ScrollText } from "lucide-react";
import { List, ListRow, PageHeader, Section, Skeleton, ThemeSelect } from "@/src/shared/ui/kit";
import { Link } from "@/src/shared/lib/router";
import { LearningSettingsForm } from "@/src/features/punisher/learning-settings";
import { useAppStatus } from "@/src/shared/api/status";
import { PUNISHER_API_BASE } from "@/src/shared/api/punisher";
import { BRAND_LINE, PunisherLogo } from "@/src/shared/ui/brand";

/** 설정: 예시 코드 언어 · 화면 테마 · Claude Code 연결 상태 · 데이터 위치. */
export function SettingsPage() {
  return (
    <div className="mx-auto flex w-full max-w-2xl flex-col gap-6">
      <PageHeader title="설정" />
      <Section title="학습">
        <LearningSettingsForm />
      </Section>
      <Section title="고급">
        <List>
          <ListRow
            href="/settings/prompts"
            linkAs={Link}
            leading={<ScrollText />}
            title="강의 프롬프트"
            meta="강의를 만드는 지시문을 단계별로 보고 고쳐요"
          />
          <ListRow
            href={`${PUNISHER_API_BASE}/export`}
            leading={<Download />}
            title="데이터 내보내기"
            meta="강의·진도·기록을 파일 하나로 받아요"
          />
        </List>
      </Section>
      <Section title="화면">
        <ThemeSelect className="w-fit" />
      </Section>
      <Section title="Claude Code 연결">
        <ClaudeStatusPanel />
      </Section>
      <Section title="정보">
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
  const status = useAppStatus();
  if (status.isLoading) { return <Skeleton variant="block" className="h-20" />; }
  if (!status.data) { return <p className="text-sm text-danger">상태를 불러오지 못했어요.</p>; }
  const { claude, message, dataDir, appVersion } = status.data;
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
          {ok ? "연결됨 — 강의 생성·질문에 내 Claude Code 계정을 사용해요." : message}
        </span>
      </p>
      <dl className="grid grid-cols-[auto_1fr] gap-x-4 gap-y-1 text-fg-secondary">
        <dt>Claude Code</dt>
        <dd className="text-fg">{claude.installed ? claude.version || "설치됨" : "설치되지 않음"}</dd>
        <dt>로그인</dt>
        <dd className="text-fg">{claude.loggedIn === null ? "확인 불가(구버전)" : claude.loggedIn ? "됨" : "안 됨"}</dd>
        <dt>진행 중 생성</dt>
        <dd className="text-fg">
          {claude.active}개{claude.queued > 0 ? ` · 대기 ${claude.queued}개` : ""}
        </dd>
        <dt>데이터 위치</dt>
        <dd className="break-all font-mono text-xs text-fg">{dataDir}</dd>
        <dt>Punisher 버전</dt>
        <dd className="text-fg">{appVersion}</dd>
      </dl>
    </div>
  );
}
