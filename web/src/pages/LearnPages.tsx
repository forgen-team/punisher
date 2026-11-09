import { Suspense } from "react";
import { useParams } from "react-router-dom";
import { EmptyState, ListSkeleton, Skeleton } from "@/src/shared/ui/kit";
import { Link } from "@/src/shared/lib/router";
import { TodayWidget } from "@/src/widgets/punisher/today";
import { KnowledgeMapWidget } from "@/src/widgets/punisher/knowledge-map";
import { HistoryWidget } from "@/src/widgets/punisher/history";
import { LectureViewer } from "@/src/widgets/punisher/lecture-viewer";
import { InstantStreamView } from "@/src/widgets/punisher/instant-stream";

export function TodayPage() {
  return <TodayWidget />;
}

export function MapPage() {
  return (
    <Suspense fallback={<Skeleton variant="block" className="h-96" />}>
      <KnowledgeMapWidget />
    </Suspense>
  );
}

export function HistoryPage() {
  return (
    <Suspense fallback={<ListSkeleton rows={6} />}>
      <HistoryWidget />
    </Suspense>
  );
}

export function LecturePage() {
  const { id = "" } = useParams();
  return <LectureViewer key={id} lectureId={id} />;
}

export function InstantPage() {
  const { jobId = "" } = useParams();
  return <InstantStreamView key={jobId} jobId={jobId} />;
}

export function NotFoundPage() {
  return (
    <EmptyState
      title="페이지를 찾을 수 없어요"
      description={
        <Link href="/" className="underline">
          오늘의 강의로 돌아가기
        </Link>
      }
    />
  );
}
