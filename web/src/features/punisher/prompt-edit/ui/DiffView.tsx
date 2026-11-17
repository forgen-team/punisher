
import { useMemo } from "react";
import { cn } from "@/src/shared/ui/kit";
import { useT } from "@/src/shared/i18n";
import { changedLines, lineDiff } from "../lib/lineDiff";
import { messages } from "../i18n";

const PREFIX = { same: "", add: "+", del: "−" } as const;

/** 기본값 → 지금 편집 중인 본문의 줄 단위 비교(읽기 전용). */
export function DiffView({ before, after }: { before: string; after: string }) {
  const diff = useMemo(() => lineDiff(before, after), [before, after]);
  const changed = changedLines(diff);
  const t = useT(messages);
  return (
    <div className="flex flex-col gap-2">
      <p className="text-xs text-fg-muted">
        {changed === 0 ? t.diffSame : t.diffChanged(changed)}
      </p>
      {/* 같으면 본문을 한 벌 더 보이지 않는다. 줄머리(+ · −)는 따로 열에 둬 줄이 접혀도 글자 시작이 맞는다. */}
      {changed > 0 && (
        <pre className="max-h-[60vh] overflow-auto rounded-md border border-line bg-canvas-subtle py-2 text-xs leading-5">
          {diff.map((line, index) => (
            <div
              key={index}
              className={cn(
                "grid grid-cols-[1rem_minmax(0,1fr)] gap-x-1 px-3",
                line.kind === "add" && "bg-success-subtle text-success",
                line.kind === "del" && "bg-danger-subtle text-danger",
              )}
            >
              <span aria-hidden="true" className="select-none text-fg-muted">{PREFIX[line.kind]}</span>
              <span className={cn("whitespace-pre-wrap break-words", line.kind === "del" && "line-through decoration-1")}>
                <span className="sr-only">{line.kind === "add" ? t.diffAdded : line.kind === "del" ? t.diffRemoved : ""}</span>
                {line.text || " "}
              </span>
            </div>
          ))}
        </pre>
      )}
    </div>
  );
}
