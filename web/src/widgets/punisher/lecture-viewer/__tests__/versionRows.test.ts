import { describe, expect, it } from "vitest";
import type { LinkedLecture } from "@/src/shared/api/punisher";
import { versionRows } from "../lib/versionRows";

const linked = (id: string, version: number, title: string, createdAt: string): LinkedLecture => ({
  id, conceptId: "c", conceptName: "데이터 표현", title, kind: "library", status: "published", createdAt, linkedAt: createdAt,
  anchor: "", percent: 0, linkKind: "regenerated", promptVersion: `p${version}`, version,
});

describe("regenerated version rows", () => {
  it("orders by version, marks the one being read and hides repeated titles", () => {
    const rows = versionRows(
      [linked("b", 3, "데이터 표현", "2026-11-20"), linked("a", 2, "데이터 표현", "2026-11-10"), linked("c", 4, "비트와 바이트", "2026-11-24")],
      { id: "cur", title: "데이터 표현", version: 1, createdAt: "2026-11-01", promptVersion: "p1" },
    );
    expect(rows.map((r) => [r.id, r.version, r.current, r.distinctTitle])).toEqual([
      ["cur", 1, true, ""], ["a", 2, false, ""], ["b", 3, false, ""], ["c", 4, false, "비트와 바이트"],
    ]);
  });
});
