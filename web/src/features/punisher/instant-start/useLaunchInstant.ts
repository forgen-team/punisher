import { usePathname, useRouter } from "@/src/shared/lib/router";
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

/** 열자마자 생성을 시작하는 화면(/start?concept=) — 여기서 떠날 때는 기록을 바꿔야 뒤로 가도 다시 생성하지 않는다. */
const LAUNCH_ON_OPEN_PATH = "/start";

export interface LaunchInstantOptions {
  /** 강의·생성 화면으로 갈 때 기록을 바꿀지(기본: 지금 화면이 /start 면 바꾼다) */
  replace?: boolean;
}

/**
 * Starts an instant lecture and navigates when the result is a lecture or job.
 * needs_selection / already_learned responses are returned to the caller.
 */
export function useLaunchInstant(options: LaunchInstantOptions = {}) {
  const router = useRouter();
  const pathname = usePathname();
  const mutation = useStartInstant();
  const replace = options.replace ?? pathname === LAUNCH_ON_OPEN_PATH;

  async function launch(body: InstantRequest): Promise<InstantResponse | null> {
    const res = await mutation.mutateAsync(body);
    const go = (href: string) => (replace ? router.replace(href) : router.push(href));
    if (res.status === "existing" && res.lectureId) {
      go(lectureHref(res.lectureId));
      return null;
    }
    if ((res.status === "started" || res.status === "attached") && res.jobId) {
      go(instantHref(res.jobId));
      return null;
    }
    return res;
  }

  return { launch, isPending: mutation.isPending, error: mutation.error, reset: mutation.reset };
}
