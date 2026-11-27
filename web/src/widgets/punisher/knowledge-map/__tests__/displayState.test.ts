import { describe, expect, it } from "vitest";
import { messages } from "../i18n";
import { mapDisplayState, stateLabel } from "../lib/displayState";

const base = { state: "not_started", locked: false, frontier: false } as const;

describe("지도 표시 상태 — 필요 없음", () => {
  it("추천에서 뺀 개념은 필요 없음(요청·추천·잠김보다 앞)", () => {
    expect(mapDisplayState({ ...base, dismissed: true })).toBe("dismissed");
    expect(mapDisplayState({ ...base, locked: true, dismissed: true })).toBe("dismissed");
    expect(mapDisplayState({ ...base, frontier: true, dismissed: true })).toBe("dismissed");
    expect(mapDisplayState({ ...base, dismissed: true, requests: [{ id: "r", query: "q", source: "manual", status: "unresolved" }] })).toBe("dismissed");
  });

  it("배움·읽는 중·이미 앎은 뺀 적이 있어도 그대로", () => {
    expect(mapDisplayState({ ...base, state: "completed", dismissed: true })).toBe("completed");
    expect(mapDisplayState({ ...base, state: "known", dismissed: true })).toBe("known");
    expect(mapDisplayState({ ...base, state: "in_progress", dismissed: true })).toBe("in_progress");
  });

  it("빼지 않았으면 예전과 같다", () => {
    expect(mapDisplayState(base)).toBe("not_started");
    expect(mapDisplayState({ ...base, dismissed: false, locked: true })).toBe("locked");
  });

  it("상태 이름은 ko 필요 없음 · en Not needed", () => {
    expect(stateLabel(messages.ko, "dismissed")).toBe("필요 없음");
    expect(stateLabel(messages.en, "dismissed")).toBe("Not needed");
  });
});
