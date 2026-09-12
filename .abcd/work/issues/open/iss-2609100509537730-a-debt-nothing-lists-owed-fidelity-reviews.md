---
schema_version: 1
id: "iss-2609100509537730"
slug: "a-debt-nothing-lists-owed-fidelity-reviews"
severity: "major"
category: "future-work-seed"
source: "user-observation"
found_during: "autonomous-run field experiment in a managed repository, 2026-09-09/10"
origin: researcher-authored
production_mode: hand-written
deferred_after: "v0.8.0"
deferral_reason: "Owed fidelity reviews accumulate and nothing counts them. Fixing it means deciding where the count belongs and what it should do: a number on a status board is one answer, a refusal at the cut is another, and they differ in how much a debt is allowed to block. The record's own line is the argument, that a debt nothing lists is a debt nobody pays, and it deserves a considered surface rather than a counter bolted to whichever verb was nearest."
found_at: "internal (intent audit receipts, status render, lint)"
---

A debt nothing lists is a debt nobody pays. Every shipped intent in a managed repository carries an owed fidelity review, and no surface counts them.

Observed in an autonomous run: twenty receipts owed in that repository, none ingested, and nothing in any status render, lint row or listing that says so. The obligation is real and by design — a shipped intent owes a fidelity review, that is what the receipt is for — but it accrues silently. There is no verb that answers "what is outstanding", so the only way to learn the number is to enumerate the shipped intents and check each for an ingested verdict, which is exactly the work the debt makes expensive and which nobody does unprompted.

Corroborating evidence from abcd's own repository on the same day: three receipts were owed here, and they were discharged only because a human asked what was outstanding. The tool that defines the obligation does not track it for itself either.

This is a designed obligation that silently accumulates, which is a sharper failure than a missing convenience. A debt with no counter grows until someone looks, and by then discharging it is a project rather than a step — twenty audits is a day's work, three is ten minutes. The receipts already exist as records, so the count is a directory read.

Wanted: a row in the status render or in `abcd lint` that names the number of owed fidelity reviews and the intents they belong to, and a listing verb that enumerates them. Related and filed separately: two defects that make an owed review expensive to discharge once found — the request carries no provenance hashes that `ingest` nevertheless requires, and it asks the host for a delivered diff range it has no mechanism to supply.
