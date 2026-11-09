
import { useEffect, useState } from "react";
import { anchorElementId } from "@/src/features/punisher/lecture-progress";

/** 읽는 줄(화면 위에서 30%)을 지난 마지막 섹션 — 목차의 '지금 여기' 표시용. anchors 는 '|'로 이은 문자열(안정된 의존성). */
export function useActiveSection(anchorKey: string): string {
  const [active, setActive] = useState("");
  useEffect(() => {
    const anchors = anchorKey ? anchorKey.split("|") : [];
    if (anchors.length === 0) { return; }
    let frame = 0;
    const update = () => {
      frame = 0;
      const line = window.innerHeight * 0.3;
      let current = anchors[0];
      for (const anchor of anchors) {
        const el = document.getElementById(anchorElementId(anchor));
        if (el && el.getBoundingClientRect().top <= line) { current = anchor; }
      }
      setActive((prev) => (prev === current ? prev : current));
    };
    const schedule = () => {
      if (frame === 0) { frame = requestAnimationFrame(update); }
    };
    update();
    window.addEventListener("scroll", schedule, { passive: true });
    window.addEventListener("resize", schedule);
    return () => {
      if (frame !== 0) { cancelAnimationFrame(frame); }
      window.removeEventListener("scroll", schedule);
      window.removeEventListener("resize", schedule);
    };
  }, [anchorKey]);
  return active;
}
