# Contributing to Punisher

Thanks for helping. Punisher is a small, local-only learning app: one Go binary
with a React UI embedded, one SQLite file, and the user's own Claude Code login.
The UI and the lectures are in Korean; code, comments and commit messages may be
in either language.

## 1. Set up

- Go (the version in `go.mod`) and Node 22
- [Claude Code](https://claude.ai/code) logged in (`claude auth status`) — only
  needed to generate lectures, not to run the tests

```sh
cd web && npm ci && cd ..
make test        # go vet · go test · typecheck · lint · vitest
make build       # dist/punisher (macOS universal)
make linux       # dist/punisher-linux-{amd64,arm64}
```

Run a development copy against a throwaway data folder so you never touch your
own lectures:

```sh
PUNISHER_SCHEDULER_ENABLED=false go run ./cmd/punisher --data-dir /tmp/punisher-dev --no-browser
```

CI runs `make test` on macOS and Ubuntu for every pull request.

## 2. Curriculum

The knowledge map lives in `curriculum/*.yaml`, one file per area. It is the
easiest place to contribute.

| Field | Meaning |
| --- | --- |
| `id` | `area.slug`, lowercase with hyphens (`net.tcp-handshake`). Never rename an existing id — progress is stored by id. |
| `name` | Korean display name |
| `aliases` | Other names people search for (Korean and English). Must be unique across the whole map after normalization. |
| `depth` | 1 = foundations, 2 = applied, 3 = advanced |
| `summary` | 1–2 sentences: what it is and why it matters in practice |
| `objectives` | 2–4 things a learner can *do* afterwards ("…을 설명한다", "…을 계산한다"). Lectures, checks and reviews are built from these. |
| `prerequisites` | Concept ids that must be learned first (no cycles) |
| `related`, `part_of` | Optional links for the map |
| `practical_cases` | Real situations where the concept decides something |
| `doc_links` | Optional official documentation (`title`, `url`, https only). No blogs. |

Check your change:

```sh
go test ./internal/curriculum/   # ids, aliases, prerequisites (cycles), objectives
```

### Glossary

`curriculum/glossary/<area>.yaml` holds the term dictionary: a fixed popover
gloss for every concept (`concepts: {id: text}`) and basic terms that are not
concepts (`terms:` — `id: g.<lowercase-hyphen>`, `name`, `aliases`, `concept`
for "더 깊게", `gloss`). Lectures link a term as `[SYN](term:g.syn)` instead of
re-defining it, so a term reads the same everywhere. Glosses are 2–3 plain
sentences (≤ 260 characters): definition first, then why it matters in practice;
no unconditional "always/never", no version-specific numbers. A term must not
reuse a concept's name or alias — link the concept instead.

```sh
go test ./internal/glossary/   # every concept glossed, ids/names unique, lengths
```

Changing an objective's text marks existing evidence for it as stale on users'
machines (they re-check it), so prefer adding objectives over rewording them.

## 3. Code

- The standalone app is a port of the Punisher service in a private monorepo;
  `docs/SYNC.md` describes how the two are kept in step. Pull requests here are
  welcome all the same — say so in the description if a change is
  standalone-only.
- New database columns or tables need a new numbered file in `migrations/` and an
  upgrade test from the previous release's schema (see `TestUpgradeFromV015`).
- LLM output is never trusted: parse it, validate it in Go, and record failures as
  failures. Grading moves (boxes, review dates) are pure Go functions in
  `internal/learning` with table tests.
- Lab commands shown in lectures must pass `lecture.CheckLabCommand`
  (single read-only command, allow-listed tools and flags). Extend the policy
  table with tests, never loosen it to "anything".
- Keep UI text short; no explanatory sentences in headers or panels.

## 4. Data and privacy

Everything stays in the data folder (`~/Library/Application Support/Punisher` on
macOS, `~/.local/share/punisher` on Linux). `punisher export` writes it to one zip
— it includes what you typed, including pasted lab outputs, so check before
sharing an export in an issue. Please attach screenshots or minimal steps
instead of your database.
