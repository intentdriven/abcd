---
schema_version: 1
id: "iss-2610090821489740"
slug: "guard-steps-over-bash-init-file"
severity: "major"
category: "security"
source: "agent-finding"
found_during: "private security report, filed 2026-10-05"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/payload.go"
remedy: "In `shellReadsStream`, treat an `--init-file` or `--rcfile` value as a script the shell executes: a process substitution there returns true, and a path is a file the guard has not read, failing closed on interpreter-reads-stream; prove it with a guard test (watched fail first) that the three reproduction lines are block / interpreter-reads-stream, `bash --version` and `bash -i -c true` stay allow and the process-substitution script stays a block; sweep siblings (other value options in shellStreamValueOptions and other shells' startup-file options)."
duplicates: [iss-2610090821484829]
resolution: "A startup file the command line selects is read before the guard judges the command (adr-2610091150447054 decision 1): a --rcfile or --init-file value is read and judged, and refused as a stream when it is a process substitution; the zsh startup files under an assigned ZDOTDIR or HOME, a login or interactive bash's under an assigned HOME, the logout files and the other shell-family members' profile and rc files are read the same way."
impact: fix
---

`abcd guard` steps over the value of `bash --init-file` and `--rcfile`, and interactive bash runs that file before `-c`, so a blocker in it runs while the checked command reads `bash -i -c true`.

A private security report, fixed in this release; its advisory, with the full text and reproduction, is published with the release.

Evidence (lines at main 7549ca2d5): `shellReadsStream` knows the two options only to skip their value (internal/core/guard/payload.go:1914, the list `shellStreamValueOptions` at :1943). The same process substitution in script position is a block through `readsScriptStream` (internal/core/guard/payload.go:1816).