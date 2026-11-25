import { describe, expect, it } from "vitest";
import type { LinkedLecture } from "@/src/shared/api/punisher";
import { latestPublished } from "../lib/newerVersion";

function link(id: string, status: string, linkedAt: string): LinkedLecture {
  return { id, conceptId: "c", conceptName: "", title: id, kind: "instant", status, createdAt: linkedAt, linkedAt, anchor: "", percent: 0, linkKind: "regenerated", promptVersion: "", version: 1 };
}

describe("latestPublished", () => {
  it("picks the newest published version and skips unpublished ones", () => {
    const got = latestPublished([
      link("old", "published", "2026-11-13T10:00:00Z"),
      link("draft", "generating", "2026-11-14T10:00:00Z"),
      link("new", "published", "2026-11-14T09:00:00Z"),
    ]);
    expect(got?.id).toBe("new");
  });
  it("returns undefined without a published version", () => {
    expect(latestPublished([link("draft", "failed", "2026-11-14T10:00:00Z")])).toBeUndefined();
  });
});
