import { snippetParts } from "../lib/hits";

/** 검색 결과 발췌 — 일치한 부분을 강조한다. */
export function Snippet({ text, q }: { text: string; q: string }) {
  return (
    <>
      {snippetParts(text, q).map((part, i) =>
        part.match ? <mark key={i} className="bg-accent-subtle px-1 text-accent-text">{part.text}</mark> : <span key={i}>{part.text}</span>)}
    </>
  );
}
