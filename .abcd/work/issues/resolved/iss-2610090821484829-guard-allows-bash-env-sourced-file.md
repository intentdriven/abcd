---
schema_version: 1
id: "iss-2610090821484829"
slug: "guard-allows-bash-env-sourced-file"
severity: "major"
category: "security"
source: "agent-finding"
found_during: "private security report, filed 2026-10-05"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/payload.go"
remedy: "Needs a decision: for the file form the successor recommends (`bash /tmp/s.sh`, `source /tmp/s.sh`), either the guard refuses a shell whose script operand is a path it has not read, or the successor stops recommending it and the brief names it a residual; then block a `BASH_ENV=` prefix (and `ENV=` on `bash --posix`) on a shellFamily shell as interpreter-reads-stream even when the -c payload is harmless, proved by a guard test (watched fail first) that `BASH_ENV=/tmp/e bash -c true` and the `--noprofile --norc` form block, `bash -c true` and `bash -c 'git status'` stay allow and both file forms pin the chosen verdict; sweep siblings (every startup-file variable a shellFamily shell honours)."
resolution: "The guard reads a script a shell runs before it judges the command (adr-2610091150447054): a script operand, a source operand, BASH_ENV and ENV are read and judged with the registry's command-position matches, a script written earlier on the same line is refused, and the stream refusal's successor no longer recommends an unread file."
impact: fix
---

`abcd guard` allows `BASH_ENV=<file> bash -c true`, and non-interactive bash sources that file before `-c`, so a blocker written to the file in the same command runs.

A private security report, fixed in this release; its advisory, with the full text and reproduction, is published with the release.

Evidence (lines at main 7549ca2d5): `shellReadsStream` treats a `-c` string as the payload and does not look at an assignment prefix for `BASH_ENV` (internal/core/guard/payload.go:1860). The stream block's own successor tells the caller "to run a script, save it and run it as a file after reading it" (internal/core/guard/payload.go:1971), and the file form it recommends is also an allow that bash runs, so closing `BASH_ENV` alone leaves the file channel open.