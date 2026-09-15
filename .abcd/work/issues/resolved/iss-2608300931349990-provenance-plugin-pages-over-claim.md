---
schema_version: 1
id: "iss-2608300931349990"
slug: "provenance-plugin-pages-over-claim"
severity: "minor"
category: "documentation"
source: "impl-review"
found_during: "itd-178 adversarial ruthless review, 2026-08-30"
found_at: "commands/capture.md, commands/intent.md, internal/README.md"
resolution: "The two plugin pages narrow their claim to intent, spec and issue records — the families the write paths stamp — and the sentence asserting the input assembler's field projection is deleted from both, since that exclusion is spc-56 out-of-scope and lives in code this tree does not carry. internal/README.md gains the core/provenance leaf bullet beside its sibling leaves, saying why it is a leaf."
impact: internal
resolved_by:
  intent: "itd-178"
  spec: "spc-56"
---

itd-178 prose findings: the plugin pages say every record a command writes carries the two keys, but capture disposition writes a dsp record with neither (narrow to intent, spec and issue records, as spc-56 scoped); the same pages assert the keys are excluded from every reading by the input assembler's field projection, describing code not in this tree (the sentence lives in spc-56's out-of-scope list — delete it from the user-facing surface); the internal package map omits the new provenance leaf while listing its sibling leaves.
