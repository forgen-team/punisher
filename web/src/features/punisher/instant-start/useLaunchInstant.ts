
import { useRouter } from "@/src/shared/lib/router";
import { lectureHref, useStartInstant } from "@/src/entities/punisher";
import type { InstantRequest, InstantResponse } from "@/src/shared/api/punisher";

export { lectureHref };

/** '이 질문에 답하기' 해설 화면. */
export function inquiryHref(id: string): string {
  return `/inquiries/${encodeURIComponent(id)}`;
}

export function instantHref(jobId: string): string {
  return `/instant/${encodeURIComponent(jobId)}`;
}

/**
 * Starts an instant lecture and navigates when the result is a lecture or job.
 * needs_selection / already_learned responses are returned to the caller.
 */
export function useLaunchInstant() {
  const router = useRouter();
  const mutation = useStartInstant();

  async function launch(body: InstantRequest): Promise<InstantResponse | null> {
    const res = await mutation.mutateAsync(body);
    if (res.status === "existing" && res.lectureId) {
      router.push(lectureHref(res.lectureId));
      return null;
    }
    if ((res.status === "started" || res.status === "attached") && res.jobId) {
      router.push(instantHref(res.jobId));
      return null;
    }
    return res;
  }

  return { launch, isPending: mutation.isPending, error: mutation.error, reset: mutation.reset };
}
