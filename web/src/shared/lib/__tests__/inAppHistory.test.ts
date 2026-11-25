import { describe, expect, it } from "vitest";
import { nextNavStack, type NavStack } from "../in-app-history";

const start = (entry: string): NavStack => ({ entries: [entry], index: 0 });

describe("in-app history stack (← 판정)", () => {
  it("can go back after a push and not after returning to the first screen", () => {
    let s = start("/lectures/1");
    s = nextNavStack(s, "push", "/map");
    expect(s.index).toBe(1);
    s = nextNavStack(s, "pop", "/lectures/1");
    // 예전 카운터는 여기서도 2 라 ← 가 앱 밖으로 나갔다.
    expect(s.index).toBe(0);
    s = nextNavStack(s, "pop", "/map");
    expect(s.index).toBe(1);
  });

  it("replace changes the current entry only", () => {
    let s = nextNavStack(start("/"), "push", "/map?concept=a");
    s = nextNavStack(s, "replace", "/map");
    expect(s).toEqual({ entries: ["/", "/map"], index: 1 });
  });

  it("a push after going back drops the forward entries", () => {
    let s = nextNavStack(nextNavStack(start("/"), "push", "/a"), "push", "/b");
    s = nextNavStack(s, "pop", "/a");
    s = nextNavStack(s, "push", "/c");
    expect(s).toEqual({ entries: ["/", "/a", "/c"], index: 2 });
  });

  it("an unknown shell move starts over (← goes to the default parent)", () => {
    const s = nextNavStack(nextNavStack(start("/"), "push", "/a"), "pop", "/elsewhere");
    expect(s).toEqual({ entries: ["/elsewhere"], index: 0 });
  });
});
