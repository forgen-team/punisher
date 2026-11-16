# Security

Punisher runs only on your own computer. It listens on `127.0.0.1`; phone access
(`punisher phone on`) publishes it inside your own Tailscale network only, and
requests from other Tailscale users are refused. It has no accounts, no API keys
of its own, and sends lecture prompts only through your local Claude Code CLI.

## Reporting a vulnerability

Please report security issues privately through GitHub:
**Security → Report a vulnerability** on this repository. Do not open a public
issue. Include the version (`punisher --version`), your OS, and steps to
reproduce. You will get a reply within a week.

## Things that are in scope

- Anything that lets another device or user read or change your data folder
- A lecture's `@@lab` command that passes `lecture.CheckLabCommand` but can write
  files, run arbitrary code or reveal secrets
- Path traversal or overwrite through `punisher import`
- Prompt injection that makes the app run commands (the app never runs lab
  commands itself; it only shows them)
