---
id: itd-2611181134451249
slug: five-sentences-in-internal-core-surface
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
related_issues: [iss-2610251559136898]
origin: extracted-from-record
production_mode: hand-written
---

# Five sentences in internal/core/surface/sentences.go name less than the verb writes; each renders on the verb's help and on docs/reference/cli/commands.md. (1) ahoy uninstall (sentences.go:47; page 158) says 'Writes the removal of the marker block, PATH copy, and provenance record'. Uninstall also removes the README parts (internal/core/ahoy/apply.go:2196, readme_apply.go:454) and restores the harness's status-line command (apply.go:2201, statusline_apply.go:280-310). (2) capture resolve and (3) capture wontfix (sentences.go:88, 92; page 672, 714) say 'Writes the moved record'. The transition also repoints every relative markdown link in the working tree that named the old path (internal/core/capture/workflow.go:715; internal/core/relink/relink.go:1-4). (4) build next (sentences.go:63; page 411) leaves out the reservation its start pushes to origin (internal/core/implement/loop/loop.go:412), which build's own sentence names. (5) docs fidelity (sentences.go:125; page 1141) says 'Writes drafted edits only with --apply', yet --autonomous applies them too (internal/surface/cli/docfidelity.go:128). Same class as iss-2610160608444348, which holds the drain sentence.

## Press Release

> _Seeded by promotion from iss-2610251559136898. Expand into the full press-release narrative before planning._

## Why This Matters

Graduated from `iss-2610251559136898`: Five sentences in internal/core/surface/sentences.go name less than the verb writes; each renders on the verb's help and on docs/reference/cli/commands.md. (1) ahoy uninstall (sentences.go:47; page 158) says 'Writes the removal of the marker block, PATH copy, and provenance record'. Uninstall also removes the README parts (internal/core/ahoy/apply.go:2196, readme_apply.go:454) and restores the harness's status-line command (apply.go:2201, statusline_apply.go:280-310). (2) capture resolve and (3) capture wontfix (sentences.go:88, 92; page 672, 714) say 'Writes the moved record'. The transition also repoints every relative markdown link in the working tree that named the old path (internal/core/capture/workflow.go:715; internal/core/relink/relink.go:1-4). (4) build next (sentences.go:63; page 411) leaves out the reservation its start pushes to origin (internal/core/implement/loop/loop.go:412), which build's own sentence names. (5) docs fidelity (sentences.go:125; page 1141) says 'Writes drafted edits only with --apply', yet --autonomous applies them too (internal/surface/cli/docfidelity.go:128). Same class as iss-2610160608444348, which holds the drain sentence.. Read that issue record for the source observation.

## Mechanism

> _Prompted (the claim-recording gradient): why the authors expect this to work, as a falsifiable "we expect X because Y" — not the outcome restated. Replace this line with the claim, or with the exact token `None stated.` alone on its line to record the claim as considered and declined._

## Scope Conditions

> _Required (the claim-recording gradient): the population, platform, scale, or assumptions this claim holds under, one per top-level bullet — `abcd intent plan` stamps each with a persistent identity. Replace this line with those bullets, or with the exact token `None stated.` alone on its line._

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._

## Open Questions

_None recorded yet._

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
