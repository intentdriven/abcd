---
schema_version: 1
id: "iss-2609261604498137"
slug: "the-ahoy-install-changed-line-reaches"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review-itd63"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/tools/install.go"
resolution: "A program's output becomes result text only through tools.outputLines, which passes each line through termsafe, so the ahoy changed: and note: lines and --json carry no escape, C1 control or bidi override from it."
impact: fix
resolved_by:
  commit: "291410bb"
---

The ahoy install 'changed:' line reaches the terminal unsanitised, and itd-63 put external process output on it: the verify program's first line (and a failed step's output tail) enters tools.Result.Output raw, flows through Summary() into the ahoy result's changes and notes, and cli.go prints the changes without termsafe, so an escape sequence or bidi control in a program's output rewrites the report.

## Grounds

- pursued: every render of a tools result carries the program's text with no control rune; a Summary or Output that still holds ESC, U+009B or a bidi override after a hostile program's output would show it wrong
