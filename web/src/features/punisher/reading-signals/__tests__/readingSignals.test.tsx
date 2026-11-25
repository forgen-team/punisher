import { act, render } from "@testing-library/react";
import { useRef } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { useReadingSignals } from "@/src/shared/lib/readingSignals";
import { ReadingSignalsProvider } from "../ReadingSignalsProvider";
import { SignalBuffer, focusedAnchor } from "../lib/buffer";

const post = vi.fn((..._args: unknown[]) => Promise.resolve());
vi.mock("@/src/shared/api/punisher", () => ({ punisherApi: { postReadingSignals: (...args: unknown[]) => post(...args) } }));

describe("읽기 신호 모으기", () => {
  it("keeps per-block dwell and term opens until drained", () => {
    const b = new SignalBuffer();
    b.dwell("principles-1", 1000);
    b.dwell("principles-1", 1000);
    b.dwell("principles-2", 400);
    b.term("principles-2", "db.joins");
    b.term("principles-2", "db.joins");
    expect(b.drain()).toEqual([
      { anchor: "principles-1", dwellMs: 2000 },
      { anchor: "principles-2", dwellMs: 400, terms: { "db.joins": 2 } },
    ]);
    expect(b.drain()).toEqual([]);
  });

  it("gives the time to the most visible block", () => {
    expect(focusedAnchor(new Map([["a-1", 0.4], ["a-2", 0.9]]))).toBe("a-2");
    expect(focusedAnchor(new Map([["a-1", 0.1]]))).toBeNull();
  });
});

describe("ReadingSignalsProvider", () => {
  let callback: IntersectionObserverCallback | null = null;
  beforeEach(() => {
    vi.useFakeTimers();
    post.mockClear();
    vi.stubGlobal("IntersectionObserver", class {
      constructor(cb: IntersectionObserverCallback) { callback = cb; }
      observe() {}
      disconnect() {}
    });
  });
  afterEach(() => {
    vi.useRealTimers();
    vi.unstubAllGlobals();
  });

  function Term() {
    const signals = useReadingSignals();
    return <button type="button" onClick={() => signals?.termOpened("db.joins", "principles-1")}>term</button>;
  }

  function Screen() {
    const ref = useRef<HTMLDivElement>(null);
    return (
      <div ref={ref}>
        <ReadingSignalsProvider lectureId="lec" containerRef={ref}>
          <div data-lecture-anchor="principles">
            <div data-lecture-anchor="principles-1">본문</div>
          </div>
          <Term />
        </ReadingSignalsProvider>
      </div>
    );
  }

  it("sends the focused block's dwell and the term opens every 30 seconds and on leaving", () => {
    const view = render(<Screen />);
    const block = view.container.querySelector("[data-lecture-anchor='principles-1']")!;
    act(() => {
      callback?.([{ target: block, isIntersecting: true, intersectionRatio: 1, intersectionRect: { height: 100 }, rootBounds: { height: 800 } } as unknown as IntersectionObserverEntry], {} as IntersectionObserver);
      view.getByText("term").click();
      vi.advanceTimersByTime(30_000);
    });
    expect(post).toHaveBeenCalledTimes(1);
    const [lectureId, batch, keepalive] = post.mock.calls[0];
    expect(lectureId).toBe("lec");
    expect(keepalive).toBe(false);
    expect(batch).toEqual([{ anchor: "principles-1", dwellMs: 29_000, terms: { "db.joins": 1 } }]);
    act(() => { vi.advanceTimersByTime(5000); });
    view.unmount();
    expect(post).toHaveBeenCalledTimes(2);
    expect(post.mock.calls[1][2]).toBe(true);
  });
});
