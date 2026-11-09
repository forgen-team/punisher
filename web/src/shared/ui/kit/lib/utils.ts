import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";
import type * as React from "react";

/** Merge class names with Tailwind-aware deduplication. */
export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

/** 공통 포커스 링 — 모든 인터랙티브 프리미티브가 공유한다. */
export const focusRing =
  "outline-none focus-visible:ring-2 focus-visible:ring-focus focus-visible:ring-offset-2 focus-visible:ring-offset-canvas";

/** 여러 ref(콜백/객체)를 하나로 합친다. */
export function mergeRefs<T>(...refs: Array<React.Ref<T> | undefined>) {
  return (node: T | null) => {
    for (const ref of refs) {
      if (typeof ref === "function") ref(node);
      else if (ref) (ref as React.MutableRefObject<T | null>).current = node;
    }
  };
}
