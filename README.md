<p align="center">
  <img src="assets/banner.png" alt="Punisher" width="100%"/>
</p>

<p align="center">
  <strong>Thirty minutes of CS fundamentals a day, written for you by Claude Code.</strong><br/>
  Runs on your Mac only, and generates lectures with your own Claude Code login.
</p>

<p align="center">
  <a href="LICENSE"><img src="https://img.shields.io/badge/License-MIT-yellow.svg" alt="License: MIT"/></a>
  <img src="https://img.shields.io/badge/macOS-universal-lightgrey.svg" alt="macOS universal"/>
  <a href="https://github.com/wooo-jin/forgen"><img src="https://img.shields.io/badge/forgen-family-f97316.svg" alt="Part of the forgen family"/></a>
</p>

<p align="center">
  English &middot; <a href="README.ko.md">한국어</a>
</p>

---

## What it is

Punisher is a personal learning app for computer-science fundamentals. The UI is in Korean.

- **Today's lecture.** It recommends the next concept to study and writes a ~30-minute lecture on it, streamed live as Claude writes.
- **Read actively.** Tap a term for a short gloss, select a passage for "explain this part", ask questions, take notes. "Go deeper" lectures stay linked under the lecture you came from.
- **Knowledge map.** 168 concepts across 8 areas (data structures, computer architecture, OS, network/web, databases, distributed systems, infra/ops, security/design) with your progress and goals.
- **Study requests.** Write down a topic; Punisher finds the concept and prepares its lecture in advance.
- **Your prompts.** Every instruction block (outline → lecture → review → gloss → Q&A → explain) can be viewed and edited, with preview and history.
- **Local only.** One binary serves the web UI on `127.0.0.1`, stores everything in one SQLite file, and calls the `claude` CLI you are already logged in to. No server, Docker, database, API key or account.

## Requirements

- macOS (Apple Silicon or Intel)
- [Claude Code](https://docs.claude.com/en/docs/claude-code) installed and logged in:
  ```sh
  curl -fsSL https://claude.ai/install.sh | bash
  claude          # run once, log in, then /exit
  ```
  `claude auth status` should report `"loggedIn": true`. Usage counts against your Claude subscription.

## Install

1. Unzip `punisher-<version>-macos-universal.zip`.
2. In a terminal, from the unzipped folder:
   ```sh
   sh install.sh      # copies punisher to ~/bin and adds ~/bin to PATH
   ```
   Or by hand:
   ```sh
   mkdir -p ~/bin && cp punisher ~/bin/ && chmod +x ~/bin/punisher
   xattr -d com.apple.quarantine ~/bin/punisher   # unsigned binary: skip the Gatekeeper prompt
   ```

## Run

```sh
punisher                 # opens http://127.0.0.1:4710 (or a free port)
punisher --no-browser    # don't open the browser
punisher --port 4800     # pick a port
punisher --version
```

Stop it with `Ctrl+C`. Running it again while it is up just reopens the page.

Data lives in `~/Library/Application Support/Punisher/punisher.db` (one SQLite file — copy the folder to back it up). Upgrades migrate it automatically; see [CHANGELOG.md](CHANGELOG.md).

## Build from source

```sh
make build     # web UI (Vite + React) embedded into a universal macOS binary: dist/punisher
make test      # go vet + go test + typecheck + lint + vitest
make release VERSION=v0.2.0   # dist/release/punisher-v0.2.0-macos-universal.zip
```

Go 1.24+ and Node 20+. SQLite is pure Go (modernc.org/sqlite), so no CGO.
Logo and banner PNGs are generated from `assets/*.svg` with `scripts/generate-brand-assets.sh` (rsvg-convert).
Punisher is ported from a private monorepo; see [NOTICE](NOTICE) and [docs/SYNC.md](docs/SYNC.md).

## Part of the forgen family

Punisher is a companion to [forgen](https://github.com/wooo-jin/forgen), which makes Claude fit the way you work.
It shares the forgen flame and orange accent with its sibling Handoff, a local workflow agent you drive from your phone.

## License

MIT © 2026 jang-ujin
