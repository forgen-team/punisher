import { Button } from "@forgen-team/ui";
import { Link } from "@/src/shared/lib/router";
import { useT } from "@/src/shared/i18n";
import { messages } from "../i18n";

/** 빈 상태의 다음 행동 — 오늘 화면으로. */
export function ToToday() {
  const t = useT(messages);
  return (
    <Button asChild variant="secondary">
      <Link href="/">{t.toToday}</Link>
    </Button>
  );
}
