
import { ErrorState, Field, ListSkeleton, Select, Spinner } from "@/src/shared/ui/kit";
import { useLearningSettings, useSaveLearningSettings } from "@/src/entities/punisher";
import { describePunisherError, type ExampleLanguage } from "@/src/shared/api/punisher";

const DAILY_MINUTE_OPTIONS = [15, 30, 45, 60] as const;

/**
 * 학습 설정 — 예시 코드 언어 · 하루 학습량. 바꾸면 바로 저장되고, 이후 생성되는 강의·질문 답변·
 * "이 부분 설명"에 적용된다(이미 만든 강의는 다시 만들지 않는다).
 */
export function LearningSettingsForm() {
  const settings = useLearningSettings();
  const save = useSaveLearningSettings();

  if (settings.isLoading) { return <ListSkeleton rows={2} />; }
  if (settings.error || !settings.data) {
    return (
      <ErrorState
        compact
        message={describePunisherError(settings.error)}
        onRetry={() => settings.refetch()}
        retrying={settings.isRefetching}
      />
    );
  }

  const pending = save.isPending ? save.variables : undefined;
  const current = pending?.exampleLanguage ?? settings.data.settings.exampleLanguage;
  const daily = pending?.dailyMinutes ?? settings.data.settings.dailyMinutes ?? 30;

  return (
    <div className="flex flex-col gap-2">
      <div className="flex flex-col gap-4">
        <Field
          label="예시 코드 언어"
          hint="강의·질문 답변·'이 부분 설명'의 코드 예시를 이 언어로 써요. 셸 명령·SQL·설정 파일은 원래 형식 그대로예요. 이미 만든 강의는 바뀌지 않아요."
          error={save.error ? describePunisherError(save.error) : undefined}
        >
          <Select value={current} disabled={save.isPending} onChange={(event) => save.mutate({ exampleLanguage: event.target.value as ExampleLanguage })}>
            {settings.data.exampleLanguages.map((lang) => (
              <option key={lang.id} value={lang.id}>
                {lang.label}
              </option>
            ))}
          </Select>
        </Field>
        <Field
          label="하루 학습량"
          hint="남은 강의가 이 속도로 2주 치보다 적어지면 오늘 화면에서 '다음 챕터 고르기'를 띄워요."
        >
          <Select value={String(daily)} disabled={save.isPending} onChange={(event) => save.mutate({ dailyMinutes: Number(event.target.value) })}>
            {DAILY_MINUTE_OPTIONS.map((m) => (
              <option key={m} value={m}>
                하루 {m}분
              </option>
            ))}
          </Select>
        </Field>
      </div>
      <p aria-live="polite" className="flex min-h-5 items-center gap-1.5 text-xs text-fg-muted">
        {save.isPending && <Spinner size="sm" label={null} />}
        {save.isPending && "저장 중…"}
        {!save.isPending && save.isSuccess && "저장했어요."}
      </p>
    </div>
  );
}
