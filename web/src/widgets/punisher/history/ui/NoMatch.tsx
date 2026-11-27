import { Button, EmptyState } from "@forgen-team/ui";
import { useT } from "@/src/shared/i18n";
import { messages } from "../i18n";
import { useHistoryNav } from "./useHistoryNav";

/** 검색어에 맞는 기록 0건 — 조건 때문이라고 말하고 [검색어 지우기]. */
export function NoMatch() {
  const t = useT(messages);
  const { q, go } = useHistoryNav();
  return <EmptyState title={t.noMatch(q)} action={<Button variant="secondary" onClick={() => go({ q: "" })}>{t.clearQuery}</Button>} />;
}
