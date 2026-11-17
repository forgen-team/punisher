import { describe, expect, it } from "vitest";
import { INITIAL_STAGE, currentStep, failureLines, reduceStage, stageLabel, type StageState } from "../lib/stageState";

function run(events: Array<[Parameters<typeof reduceStage>[1], Record<string, unknown> | null]>): StageState {
  return events.reduce((s, [name, data]) => reduceStage(s, name, data), INITIAL_STAGE);
}

describe("reduceStage — 새 stage 이벤트", () => {
  it("walks writing → checking → reviewing → rewriting n/2 → published", () => {
    const writing = run([["stage", { type: "stage", stage: "writing", attempt: 0, maxAttempts: 2 }]]);
    expect(stageLabel(writing, "ko")).toBe("작성 중");
    expect(currentStep(writing.stage)).toBe("write");
    const reviewing = reduceStage(writing, "stage", { stage: "reviewing" });
    expect(stageLabel(reviewing, "ko")).toBe("검수 중");
    expect(stageLabel(reduceStage(writing, "stage", { stage: "checking" }), "ko")).toBe("검수 중");
    const rewriting = reduceStage(reviewing, "stage", { stage: "rewriting", attempt: 1, maxAttempts: 2, reasons: ["Θ 혼동"] });
    expect(stageLabel(rewriting, "ko")).toBe("고치는 중 1/2");
    expect(rewriting.reasons).toEqual(["Θ 혼동"]);
    const published = reduceStage(rewriting, "stage", { stage: "published", lectureId: "L1" });
    expect(published.stage).toBe("published");
    expect(published.lectureId).toBe("L1");
  });

  it("accepts the stage payload on the default message event", () => {
    expect(run([["message", { type: "stage", stage: "checking" }]]).stage).toBe("checking");
  });

  it("counts rewrites when attempt is missing", () => {
    const s = run([["stage", { stage: "rewriting" }], ["stage", { stage: "reviewing" }], ["stage", { stage: "rewriting" }]]);
    expect(stageLabel(s, "ko")).toBe("고치는 중 2/2");
  });

  it("fails with up to three reason lines", () => {
    const s = run([["stage", { stage: "failed", reasons: ["a", "b", "c", "d"] }]]);
    expect(s.stage).toBe("failed");
    expect(failureLines(s)).toEqual(["a", "b", "c"]);
  });

  it("treats published without lectureId as failed and ignores unknown stages", () => {
    expect(run([["stage", { stage: "published" }]]).stage).toBe("failed");
    expect(run([["stage", { stage: "drafting" }]])).toBe(INITIAL_STAGE);
  });

  it("does not move after a final state", () => {
    const done = run([["stage", { stage: "published", lectureId: "L" }]]);
    expect(reduceStage(done, "stage", { stage: "writing" })).toBe(done);
  });
});

describe("reduceStage — 예전 이벤트 호환", () => {
  it("maps status/delta/done", () => {
    expect(run([["delta", { text: "x" }]]).stage).toBe("writing");
    expect(run([["status", { stage: "generating" }]]).stage).toBe("writing");
    expect(run([["status", { stage: "validating" }]]).stage).toBe("checking");
    const rewriting = run([["status", { stage: "rewriting", reason: "형식" }]]);
    expect(stageLabel(rewriting, "ko")).toBe("고치는 중 1/2");
    expect(rewriting.reasons).toEqual(["형식"]);
    expect(run([["done", { lectureId: "L2" }]]).lectureId).toBe("L2");
  });

  it("maps error payloads to failed with the message", () => {
    const s = run([["error", { code: "BUDGET_EXCEEDED", message: "예산 초과" }]]);
    expect(s.stage).toBe("failed");
    expect(failureLines(s)).toEqual(["예산 초과"]);
  });

  it("ignores events without payload", () => {
    expect(reduceStage(INITIAL_STAGE, "stage", null)).toBe(INITIAL_STAGE);
  });
});
