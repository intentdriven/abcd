---
schema_version: 1
id: "iss-2609251734069878"
slug: "reference-usage-names-refused-bare-form"
severity: "nitpick"
category: "documentation"
source: "impl-review"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "docs/reference/cli/commands.md"
resolution: "The reference generator gives a command whose bare form moved a usage of its sub-verb form plus the bare form's successor; TestReferenceUsageNeverOffersAMovedBareForm holds every such command to it."
impact: fix
resolved_by:
  commit: "9b828076"
---

The CLI reference gives abcd ahoy remote and abcd identity a Usage line of their bare spelling, which only refuses: the reference names the new forms only everywhere else, so a reader copying the Usage line runs a refusal rather than the sub-verb form or the successor the bare form names

## Grounds

- pursued: no Usage line in the CLI reference offers an invocation that only refuses; shown wrong by a moved bare form whose section still gives its bare spelling, which the tree walk would name
