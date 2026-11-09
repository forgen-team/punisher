
import { ErrorState, Field, ListSkeleton, Select, Spinner } from "@/src/shared/ui/kit";
import { useLearningSettings, useSaveLearningSettings } from "@/src/entities/punisher";
import { describePunisherError, type ExampleLanguage } from "@/src/shared/api/punisher";

/**
 * 학습 설정 — 예시 코드 언어. 바꾸면 바로 저장되고, 이후 생성되는 강의·질문 답변·
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

  const current = save.isPending && save.variables ? save.variables : settings.data.settings.exampleLanguage;

  return (
    <div className="flex flex-col gap-2">
      <Field
        label="예시 코드 언어"
        hint="강의·질문 답변·'이 부분 설명'의 코드 예시를 이 언어로 써요. 셸 명령·SQL·설정 파일은 원래 형식 그대로예요. 이미 만든 강의는 바뀌지 않아요."
        error={save.error ? describePunisherError(save.error) : undefined}
      >
        <Select value={current} disabled={save.isPending} onChange={(event) => save.mutate(event.target.value as ExampleLanguage)}>
          {settings.data.exampleLanguages.map((lang) => (
            <option key={lang.id} value={lang.id}>
              {lang.label}
            </option>
          ))}
        </Select>
      </Field>
      <p aria-live="polite" className="flex min-h-5 items-center gap-1.5 text-xs text-fg-muted">
        {save.isPending && <Spinner size="sm" label={null} />}
        {save.isPending && "저장 중…"}
        {!save.isPending && save.isSuccess && "저장했어요. 다음 강의부터 적용돼요."}
      </p>
    </div>
  );
}
