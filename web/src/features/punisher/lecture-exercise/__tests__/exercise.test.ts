import { describe, expect, it } from "vitest";
import { blockContentHash, sha1Hex } from "../lib/contentHash";
import { labForOS, parseLab } from "../lib/lab";

describe("blockContentHash", () => {
  it("is sha1 (known vectors)", () => {
    expect(sha1Hex("")).toBe("da39a3ee5e6b4b0d3255bfef95601890afd80709");
    expect(sha1Hex("abc")).toBe("a9993e364706816aba3e25717850c26c9cd0d89d");
    expect(sha1Hex("a".repeat(1000))).toBe("291e9a6c66994949b57ba5e650361e98fc36b1ba");
  });
  it("matches the server's BlockContentHash (UTF-8, NUL separator)", () => {
    expect(blockContentHash("4KB 페이지 3장은 몇 바이트?", "12288 바이트")).toBe("c4ecadf8bc74");
  });
});

describe("parseLab", () => {
  it("reads the command, 보는 곳 and 예상 lines", () => {
    expect(parseLab("$ ss -tlnp\n보는 곳: Local Address 열\n예상: LISTEN 0 4096 0.0.0.0:22")).toEqual({
      command: "ss -tlnp",
      look: "Local Address 열",
      expect: "LISTEN 0 4096 0.0.0.0:22",
    });
  });
});

describe("labForOS", () => {
  it("shows any and the reader's own system", () => {
    expect(labForOS("any", "linux")).toBe(true);
    expect(labForOS("linux", "linux")).toBe(true);
    expect(labForOS("macos", "linux")).toBe(false);
    expect(labForOS("linux", "none")).toBe(false);
    expect(labForOS("macos", "none")).toBe(true);
  });
});
