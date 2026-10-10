---
schema_version: 1
id: "iss-2610090932257904"
slug: "guard-allows-shell-stdin-redirect-script"
severity: "major"
category: "security"
source: "user-observation"
found_during: "security-drain-2026-10-09 lane G sibling sweep (ADR research, reproduced at 016c1510a)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/tokenize.go"
remedy: "Under the rule A ADR, treat a < redirect into a shellFamily shell with no -c string and no script operand as the shell naming that file as its script: read and judge it with the same refusals as `bash f`, proved by a guard test watched fail first."
resolution: "A shell with no -c string and no script operand that a redirection points at a file reads that file as its script, and the guard reads and judges it the same way (adr-2610091150447054 decision 1): a < or 0< redirect, a descriptor duplicated onto stdin from a file opened earlier on the line, an exec that redirects the shell's own stdin, and the stdin of the shell that runs a string."
impact: fix
---

abcd guard allows a shell that reads its script from a file through a stdin redirect (`bash < f`, `bash -s < f`), while the pipe form `cat f | bash` blocks under interpreter-reads-stream: stdinStream (internal/core/guard/tokenize.go:49-54) is set for a pipe, a here-document, a here-string and a pipe into a group, never for a < redirect, so shellReadsStream (payload.go:1871-1874) never sees the file as the script. Reproduced at main 016c1510a. Sibling of iss-2610090821484829, found in the security-drain-2026-10-09 sweep; kept uncommitted until its fix lands.
