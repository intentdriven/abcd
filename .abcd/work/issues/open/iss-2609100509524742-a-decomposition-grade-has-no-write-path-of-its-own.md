---
schema_version: 1
id: "iss-2609100509524742"
slug: "a-decomposition-grade-has-no-write-path-of-its-own"
severity: "minor"
category: "future-work-seed"
source: "user-observation"
found_during: "autonomous-run field experiment in a managed repository, 2026-09-09/10"
origin: researcher-authored
production_mode: hand-written
found_at: "conventions (decomposition grading, calibration note)"
deferred_after: "v0.9.0"
deferral_reason: "Routed to the product thinker by the 2026-09-23 run (planning owed: a verb and a per-row store for decomposition grades). The 2026-09-23 interview gave routed minor and nitpick captures the default: deferred past v0.9.0, returning at the next anchor."
---

A decomposition grade has no write path of its own, and the note it belongs in is a single shared file, so in a parallel run the grade cannot land at all.

Observed in an autonomous run that filed, planned and implemented one intent end to end. The decomposition grading was performed by hand — there is no verb for it — and it produced a graded row that belongs in the calibration note. There is no command to file that row. Writing it by hand meant editing a file that lives on a branch another concurrent session was holding, so landing the grade would have meant a merge conflict on a note whose entire content is independent dated rows. The grade was not filed.

Two separable gaps. The grading has no command, so its output has no sanctioned destination and the format is reconstructed each time. And the destination is a monolithic file with the multi-writer merge-hotspot shape the ledger's one-file-per-record model exists to avoid — the same shape as the shared decisions log, filed separately, and with the same two candidate remedies.

Wanted: a verb that files a graded row (which also fixes the format drift), and a per-row store rather than one shared note, so a grade produced in one worktree lands without coordinating with whoever holds the branch. Until then a grade produced by a parallel session is simply lost, which is the worst outcome for a calibration record, whose value is entirely in having every row.

Related and filed separately: the grading and the planning interview are hand-run rituals, and running them repeatedly surfaced an ordering rule worth enforcing.
