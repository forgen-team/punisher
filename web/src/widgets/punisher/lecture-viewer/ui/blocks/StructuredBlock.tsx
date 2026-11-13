
import { Component, type ReactNode } from "react";
import type { LectureBlock } from "@/src/shared/api/punisher";
import { blockJson, parseChart, parseDiff, parseTrace } from "../../lib/blockData";
import { ChartBlock } from "./ChartBlock";
import { DiffBlock } from "./DiffBlock";
import { BlockError } from "./Reveal";
import { TraceBlock } from "./TraceBlock";

function Structured({ block }: { block: LectureBlock }) {
  const raw = blockJson(block);
  if (block.type === "trace") {
    const parsed = parseTrace(raw);
    return parsed.ok ? <TraceBlock data={parsed.data} /> : <BlockError />;
  }
  if (block.type === "chart") {
    const parsed = parseChart(raw);
    return parsed.ok ? <ChartBlock data={parsed.data} /> : <BlockError />;
  }
  const parsed = parseDiff(raw);
  return parsed.ok ? <DiffBlock data={parsed.data} lang={block.lang} /> : <BlockError />;
}

/** 블록 하나만 감싸는 경계 — 전역 오류 알림(토스트) 없이 이 자리만 BlockError 로 바꾼다. */
class BlockBoundary extends Component<{ children: ReactNode }, { failed: boolean }> {
  state = { failed: false };

  static getDerivedStateFromError() {
    return { failed: true };
  }

  componentDidCatch(error: unknown) {
    console.warn("punisher: structured block render failed", error);
  }

  render() {
    const { failed } = this.state;
    const { children } = this.props;
    return failed ? <BlockError /> : children;
  }
}

/** trace·chart·diff — 데이터를 검증해 그리고, 깨졌거나 그리다 실패하면 이 블록만 작은 오류 표시로 바꾼다. */
export function StructuredBlock({ block }: { block: LectureBlock }) {
  return (
    <BlockBoundary>
      <Structured block={block} />
    </BlockBoundary>
  );
}
