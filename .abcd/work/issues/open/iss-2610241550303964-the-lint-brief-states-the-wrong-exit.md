---
schema_version: 1
id: "iss-2610241550303964"
slug: "the-lint-brief-states-the-wrong-exit"
severity: "minor"
category: "documentation"
source: "agent-finding"
found_during: "hand-back triage of 2026-10-24, batch b13 reading iss-45"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/06-lint.md"
remedy: "Correct 06-lint.md:34 and :48 to say the two tree gates exit 1 on any blocker, 2 when the lint could not run and 0 otherwise, warnings included, and keep the 2/1/0 tri-state for bare abcd lint alone; proven by a CLI test that abcd lint docs over a one-blocker fixture exits 1 and over a warn-only fixture exits 0."
refines: [iss-2608260944055780]
---

The lint brief states the wrong exit codes for the two tree gates. .abcd/development/brief/05-internals/06-lint.md:48 says record-lint and abcd lint docs follow the tri-state grammar, blocker 2, warn 1, clean 0, and the severity table at line 34 says a warn finding raises the exit code to 1. The code does neither: cmd/record-lint/main.go:144 exits 1 on any blocker and 2 only when the lint could not run, internal/surface/cli/cli.go:955 makes abcd lint docs exit 1 on a blocker, and neither gate raises its exit code for warnings alone. The 2/1/0 tri-state belongs to bare abcd lint (internal/surface/cli/lint.go:76). A CI job written from the brief that keys on exit 2 for a blocker would pass a tree full of blockers.
