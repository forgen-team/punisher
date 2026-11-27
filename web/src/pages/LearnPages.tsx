import { Suspense } from "react";
import { useParams } from "react-router-dom";
import { EmptyState, ListSkeleton, Skeleton } from "@forgen-team/ui";
import { Link } from "@/src/shared/lib/router";
import { TodayWidget } from "@/src/widgets/punisher/today";
import { KnowledgeMapWidget } from "@/src/widgets/punisher/knowledge-map";
import { HistoryWidget } from "@/src/widgets/punisher/history";
import { LectureViewer } from "@/src/widgets/punisher/lecture-viewer";
import { InstantStreamView } from "@/src/widgets/punisher/instant-stream";
import { InquiryView } from "@/src/widgets/punisher/inquiry-view";
import { CheckView } from "@/src/widgets/punisher/check-view";
import { CardsWidget } from "@/src/widgets/punisher/cards";
import { SearchWidget } from "@/src/widgets/punisher/search";
import { useT } from "@/src/shared/i18n";
import { messages } from "./i18n";

/** /cards — 복습 카드(W5a). */
export function CardsPage() {
  return <CardsWidget />;
}

/** /search?q= — 학습 자료 검색(W5a). */
export function SearchPage() {
  return <SearchWidget />;
}

export function TodayPage() {
  return <TodayWidget />;
}

export function MapPage() {
  return (
    <Suspense fallback={<Skeleton lines={3} className="h-96" />}>
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

export function InquiryPage() {
  const { id = "" } = useParams();
  return <InquiryView key={id} inquiryId={id} />;
}

export function CheckPage() {
  const { id = "" } = useParams();
  return <CheckView key={id} checkId={id} />;
}

export function InstantPage() {
  const { jobId = "" } = useParams();
  return <InstantStreamView key={jobId} jobId={jobId} />;
}

export function NotFoundPage() {
  const t = useT(messages);
  return (
    <EmptyState
      title={t.notFoundTitle}
      description={
        <Link href="/" className="underline">
          {t.backToToday}
        </Link>
      }
    />
  );
}
