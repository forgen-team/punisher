import { Route } from "react-router-dom";
import { useSearchParams } from "@/src/shared/lib/router";
import { StartLectureView } from "@/src/widgets/punisher/start";

/** /start?concept=&q= — RSS 글 해설의 '이어서 배울 개념'에서 들어온다(Witems 안에서만). */
function StartPage() {
  const params = useSearchParams();
  return <StartLectureView conceptId={params.get("concept") ?? ""} query={params.get("q") ?? ""} />;
}

/** Routes that exist only inside Witems (they depend on the Witems RSS app). */
export function witemsRoutes() {
  return <Route path="start" element={<StartPage />} />;
}
