import { describe, expect, it } from "vitest";
import type { Concept, SearchHit } from "@/src/shared/api/punisher";
import { filterByKind, hitNoteKind, lectureNotesHref, matchLearnedConcepts, noteHitHref, noteHits, sortNoteRows, type NoteRow } from "../lib/notesFeed";

const row = (key: string, kind: NoteRow["kind"], at: string): NoteRow => ({ key, kind, title: key, parts: [], at });
const hit = (kind: SearchHit["kind"], patch: Partial<SearchHit> = {}): SearchHit => ({ kind, refId: "r1", lectureId: "l1", anchor: "intro", title: "", sectionTitle: "", snippet: "", ...patch });

describe("notes feed", () => {
  it("sorts kinds into one list, newest first", () => {
    const sorted = sortNoteRows([row("a", "memo", "2026-11-01"), row("b", "inquiry", "2026-11-20"), row("c", "lq", "2026-11-10")]);
    expect(sorted.map((r) => r.key)).toEqual(["b", "c", "a"]);
  });

  it("filters by kind chip; no chip keeps everything", () => {
    const rows = [row("a", "memo", "1"), row("b", "lq", "2")];
    expect(filterByKind(rows, "lq").map((r) => r.key)).toEqual(["b"]);
    expect(filterByKind(rows, "")).toHaveLength(2);
  });

  it("opens lecture rows with the notes panel and keeps ?at= first", () => {
    expect(lectureNotesHref("l1", "intro")).toBe("/lectures/l1?at=intro&panel=notes");
    expect(lectureNotesHref("l1", "")).toBe("/lectures/l1?panel=notes");
  });
});

describe("search hits on the notes tab", () => {
  it("maps hit kinds and leaves lecture-body hits to the learn tab", () => {
    expect([hit("inquiry"), hit("question"), hit("note"), hit("lecture")].map(hitNoteKind)).toEqual(["inquiry", "lq", "memo", null]);
    expect(noteHits([hit("lecture"), hit("note"), hit("question")]).map((h) => h.kind)).toEqual(["memo", "lq"]);
  });

  it("links inquiries to their screen and lecture notes to the notes panel", () => {
    expect(noteHitHref(hit("inquiry", { refId: "q9" }), "inquiry")).toBe("/inquiries/q9");
    expect(noteHitHref(hit("note"), "memo")).toBe("/lectures/l1?at=intro&panel=notes");
    expect(noteHitHref(hit("note", { lectureId: "" }), "memo")).toBeUndefined();
  });
});

describe("learned concepts", () => {
  const concept = (id: string, name: string, userState: Concept["userState"], area: string, updatedAt: string) => ({ id, name, userState, area, updatedAt }) as Concept;
  const all = [
    concept("a", "HTTP 캐시", "completed", "network", "2026-11-01"),
    concept("b", "캐시 일관성", "known", "db", "2026-11-05"),
    concept("c", "TCP", "unknown", "network", "2026-11-09"),
  ];
  it("keeps completed/known concepts that match the term and area, latest first", () => {
    expect(matchLearnedConcepts(all, "", "").map((c) => c.id)).toEqual(["b", "a"]);
    expect(matchLearnedConcepts(all, "캐시", "network").map((c) => c.id)).toEqual(["a"]);
    expect(matchLearnedConcepts(all, "tcp", "").map((c) => c.id)).toEqual([]);
  });
});
