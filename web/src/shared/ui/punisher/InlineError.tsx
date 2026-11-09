
import { describePunisherError } from "@/src/shared/api/punisher";

/** 폼·행 안의 한 줄 오류 (화면 단위 실패는 @witems/ui ErrorState). */
export function InlineError({ error }: { error: unknown }) {
  if (!error) { return null; }
  return (
    <p role="alert" className="text-sm text-danger">
      {describePunisherError(error)}
    </p>
  );
}
