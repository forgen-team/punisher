
import { ErrorState, Field, ListSkeleton, Select, Spinner } from "@/src/shared/ui/kit";
import { useLearningSettings, useSaveLearningSettings } from "@/src/entities/punisher";
import { PunisherApiError, describePunisherError, type ExampleLanguage, type LearningSettingsPatch } from "@/src/shared/api/punisher";
import { ReviewSettingsFields } from "./ReviewSettingsFields";
import type { SettingsField } from "./reviewSettings";

const DAILY_MINUTE_OPTIONS = [15, 30, 45, 60] as const;

/**
 * 학습 설정 — 예시 코드 언어 · 하루 학습량 · 숙달 기준 · 복습 간격 · 하루 복습 시간 · 복습 채점 방식.
 * 바꾸면 그 필드만 바로 저장된다(PUT /me/settings 부분 업데이트). 예시 언어는 이후 생성되는 강의·질문 답변·
 * "이 부분 설명"에 적용된다(이미 만든 강의는 다시 만들지 않는다). 저장 오류는 그 필드 옆에(400 은 서버 문구).
 */
export function LearningSettingsForm() {
  const settings = useLearningSettings();
  const save = useSaveLearningSettings();

  if (settings.isLoading) { return <ListSkeleton rows={6} />; }
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
  const shown = { ...settings.data.settings, ...pending };
  const current = shown.exampleLanguage;
  const daily = shown.dailyMinutes ?? 30;
  const saveField = (patch: LearningSettingsPatch) => save.mutate(patch);
  const fieldError = (field: SettingsField): string | undefined => {
    if (!save.error || save.variables?.[field] === undefined) { return undefined; }
    return save.error instanceof PunisherApiError && save.error.status === 400 ? save.error.message : describePunisherError(save.error);
  };

  return (
    <div className="flex flex-col gap-2">
      <div className="flex flex-col gap-4">
        <Field
          label="예시 코드 언어"
          hint="강의·질문 답변·'이 부분 설명'의 코드 예시를 이 언어로 써요. 셸 명령·SQL·설정 파일은 원래 형식 그대로예요. 이미 만든 강의는 바뀌지 않아요."
          error={fieldError("exampleLanguage")}
        >
          <Select value={current} disabled={save.isPending} onChange={(event) => saveField({ exampleLanguage: event.target.value as ExampleLanguage })}>
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
          error={fieldError("dailyMinutes")}
        >
          <Select value={String(daily)} disabled={save.isPending} onChange={(event) => saveField({ dailyMinutes: Number(event.target.value) })}>
            {DAILY_MINUTE_OPTIONS.map((m) => (
              <option key={m} value={m}>
                하루 {m}분
              </option>
            ))}
          </Select>
        </Field>
        <ReviewSettingsFields settings={shown} saving={save.isPending} onSave={saveField} fieldError={fieldError} />
      </div>
      <p aria-live="polite" className="flex min-h-5 items-center gap-1.5 text-xs text-fg-muted">
        {save.isPending && <Spinner size="sm" label={null} />}
        {save.isPending && "저장 중…"}
        {!save.isPending && save.isSuccess && "저장했어요."}
      </p>
    </div>
  );
}
