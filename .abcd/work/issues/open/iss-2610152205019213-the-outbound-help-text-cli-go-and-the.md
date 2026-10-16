---
schema_version: 1
id: "iss-2610152205019213"
slug: "the-outbound-help-text-cli-go-and-the"
severity: "nitpick"
category: "bug"
source: "user-observation"
found_during: "outbound one-switch final review, 2026-10-15"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/cli.go"
remedy: "Add one sentence on the saved copy and the chain line to the --outbound help and regenerate the reference"
---

The --outbound help text (cli.go and the generated CLI reference) does not mention that the install may save abcd's check as hooks/abcd-commit-msg and advise a chain line where git runs a hook abcd did not write; commands/ahoy.md and the brief do.
