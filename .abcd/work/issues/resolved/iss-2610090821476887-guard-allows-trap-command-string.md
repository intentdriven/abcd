---
schema_version: 1
id: "iss-2610090821476887"
slug: "guard-allows-trap-command-string"
severity: "major"
category: "security"
source: "agent-finding"
found_during: "private security report, filed 2026-10-05"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/payload.go"
remedy: "Read `trap`'s command operand the way `evalPayload` reads `eval` (drop a leading `--`, take the command word, send it back through payload expansion; an unreadable string is a block; `trap - EXIT` and `trap EXIT` stay allow); prove it with a verdict table beside the eval fixtures (watched fail first) in which every trap spelling in the reproduction is block / git-push-force, `trap - EXIT` and `trap 'echo hi' EXIT` stay allow and the DEBUG form with a following `true` blocks; sweep siblings (other builtins whose operand is a command string the shell runs later)."
resolution: "the guard reads trap's ACTION (and mapfile/readarray -C's callback) as eval's arguments are read: a readable string is judged, the reset and list forms carry no command, and text the guard cannot read keeps eval's verdict rather than the remedy's block, by the coordinator's ruling"
impact: fix
---

`abcd guard` allows a blocker written as the command string of `trap`, and bash runs it, an EXIT trap needing no further command.

A private security report, fixed in this release; its advisory, with the full text and reproduction, is published with the release.

Evidence (lines at main 7549ca2d5): `evalPayload` joins `eval`'s operands and the guard re-reads that string, which is why `eval -- 'git push --force origin main'` is a block (internal/core/guard/payload.go:1578). Nothing walks the command operand of `trap`. The hook maps an allow to exit 0 and only a block to exit 2 (internal/surface/cli/guard.go:455-470).