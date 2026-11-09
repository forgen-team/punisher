
import { useMemo } from "react";
import { cn } from "@/src/shared/ui/kit";
import { changedLines, lineDiff } from "../lib/lineDiff";

const PREFIX = { same: " ", add: "+", del: "−" } as const;

/** 기본값 → 지금 편집 중인 본문의 줄 단위 비교(읽기 전용). */
export function DiffView({ before, after }: { before: string; after: string }) {
  const diff = useMemo(() => lineDiff(before, after), [before, after]);
  const changed = changedLines(diff);
  return (
    <div className="flex flex-col gap-2">
      <p className="text-xs text-fg-muted">
        {changed === 0 ? "기본값과 같아요." : `기본값과 ${changed}줄 달라요. + 내 문구 · − 기본값에서 빠진 문구`}
      </p>
      <pre className="max-h-[60vh] overflow-auto rounded-md border border-line bg-canvas-subtle py-2 text-xs leading-5">
        {diff.map((line, index) => (
          <div
            key={index}
            className={cn(
              "whitespace-pre-wrap break-words px-3",
              line.kind === "add" && "bg-success-subtle text-success",
              line.kind === "del" && "bg-danger-subtle text-danger line-through decoration-1",
            )}
          >
            <span aria-hidden="true" className="mr-2 select-none text-fg-muted">{PREFIX[line.kind]}</span>
            <span className="sr-only">{line.kind === "add" ? "추가: " : line.kind === "del" ? "삭제: " : ""}</span>
            {line.text || " "}
          </div>
        ))}
      </pre>
    </div>
  );
}
