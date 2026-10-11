---
schema_version: 1
id: "iss-2609100509524742"
slug: "a-decomposition-grade-has-no-write-path"
severity: "minor"
category: "future-work-seed"
source: "user-observation"
found_during: "autonomous-run field experiment in a managed repository, 2026-09-09/10"
origin: researcher-authored
production_mode: hand-written
found_at: "conventions (decomposition grading, calibration note)"
deferred_after: "v0.11.1"
deferral_reason: "ruling owed to the product thinker (away; run A 2026-09-25; rulings-owed H): Plan a verb and a per-row store for decomposition grades, with iss-2609100508566700?"
remedy: "Waits on ruling (a grade verb and a per-row store, with iss-2609100508566700): if planned, add a verb that files one graded row as its own file in a per-row directory, minted through the shared record-id seam, in place of the one shared calibration note, proven by a test that two worktrees filing grades merge without conflict; if declined, wontfix naming the shared note's hand-append rule."
---

A decomposition grade has no write path of its own, and the note it belongs in is a single shared file, so in a parallel run the grade cannot land at all.

Observed in an autonomous run that filed, planned and implemented one intent end to end. The decomposition grading was performed by hand — there is no verb for it — and it produced a graded row that belongs in the calibration note. There is no command to file that row. Writing it by hand meant editing a file that lives on a branch another concurrent session was holding, so landing the grade would have meant a merge conflict on a note whose entire content is independent dated rows. The grade was not filed.

Two separable gaps. The grading has no command, so its output has no sanctioned destination and the format is reconstructed each time. And the destination is a monolithic file with the multi-writer merge-hotspot shape the ledger's one-file-per-record model exists to avoid — the same shape as the shared decisions log, filed separately, and with the same two candidate remedies.

Wanted: a verb that files a graded row (which also fixes the format drift), and a per-row store rather than one shared note, so a grade produced in one worktree lands without coordinating with whoever holds the branch. Until then a grade produced by a parallel session is simply lost, which is the worst outcome for a calibration record, whose value is entirely in having every row.

Related and filed separately: the grading and the planning interview are hand-run rituals, and running them repeatedly surfaced an ordering rule worth enforcing.

## Remedy grounds (2026-09-29)

Why: one file per row is the ledger's own answer to a multi-writer hot spot, and the record-id seam already mints without coordination between checkouts (adr-45). Rejected: a lock on the shared note, which cannot serialise writers on different branches.
