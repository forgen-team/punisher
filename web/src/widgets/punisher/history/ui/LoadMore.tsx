
import { Button } from "@forgen-team/ui";
import { useT } from "@/src/shared/i18n";
import { messages } from "../i18n";

interface LoadMoreProps {
  hasNextPage: boolean;
  isFetching: boolean;
  onLoad: () => void;
}

export function LoadMore({ hasNextPage, isFetching, onLoad }: LoadMoreProps) {
  const t = useT(messages);
  if (!hasNextPage) { return null; }
  return (
    <Button variant="ghost" size="sm" loading={isFetching} onClick={onLoad} className="self-center">
      {t.loadMore}
    </Button>
  );
}
