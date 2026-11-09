import { CheckCircle2, AlertTriangle, ScrollText } from "lucide-react";
import { List, ListRow, PageHeader, Section, Skeleton, ThemeSelect } from "@/src/shared/ui/kit";
import { Link } from "@/src/shared/lib/router";
import { LearningSettingsForm } from "@/src/features/punisher/learning-settings";
import { useAppStatus } from "@/src/shared/api/status";

/** 설정: 예시 코드 언어 · 화면 테마 · Claude Code 연결 상태 · 데이터 위치. */
export function SettingsPage() {
  return (
    <div className="mx-auto flex w-full max-w-2xl flex-col gap-6 pb-10">
      <PageHeader title="설정" subtitle="강의를 내 손에 익은 언어로 읽어요." />
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
            meta="개요 짜기 → 본문 쓰기 → 감수 → 용어 설명 → 질문 답변 → 구절 설명 지시문을 보고 고쳐요"
          />
        </List>
      </Section>
      <Section title="화면">
        <ThemeSelect className="w-fit" />
      </Section>
      <Section title="Claude Code 연결">
        <ClaudeStatusPanel />
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
