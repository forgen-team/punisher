
import { Button } from "@/src/shared/ui/kit";

interface LoadMoreProps {
  hasNextPage: boolean;
  isFetching: boolean;
  onLoad: () => void;
}

export function LoadMore({ hasNextPage, isFetching, onLoad }: LoadMoreProps) {
  if (!hasNextPage) { return null; }
  return (
    <Button variant="ghost" size="sm" loading={isFetching} onClick={onLoad} className="self-center">
      더 보기
    </Button>
  );
}
